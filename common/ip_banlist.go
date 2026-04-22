package common

import (
	"bufio"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"
	"wwfc/logging"
)

type ipBanEntry struct {
	label string
	from  netip.Addr
	to    netip.Addr
}

type ipBanList struct {
	mutex       sync.RWMutex
	path        string
	lastModTime time.Time
	lastSize    int64
	lastError   string
	entries     []ipBanEntry
}

var globalIPBanList ipBanList
var ipBanListFilepath = "./ip_banlist.txt"

func IsIPBanned(remoteAddr string) (bool, string) {
	return globalIPBanList.isBanned(ipBanListFilepath, remoteAddr)
}

func (l *ipBanList) isBanned(path string, remoteAddr string) (bool, string) {
	addr, ok := parseRemoteIP(remoteAddr)
	if !ok {
		return false, ""
	}

	l.ensureLoaded(path)

	l.mutex.RLock()
	entries := l.entries
	l.mutex.RUnlock()

	for _, entry := range entries {
		if addr.BitLen() != entry.from.BitLen() {
			continue
		}

		if entry.from.Compare(addr) <= 0 && addr.Compare(entry.to) <= 0 {
			return true, entry.label
		}
	}

	return false, ""
}

func parseRemoteIP(remoteAddr string) (netip.Addr, bool) {
	host := remoteAddr

	if addrPort, err := netip.ParseAddrPort(remoteAddr); err == nil {
		return addrPort.Addr().Unmap(), true
	}

	if splitHost, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = splitHost
	}

	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}

	return addr.Unmap(), true
}

func (l *ipBanList) ensureLoaded(path string) {
	info, err := os.Stat(path)
	if err != nil {
		l.setError(path, err.Error())
		return
	}

	l.mutex.RLock()
	unchanged := l.path == path && l.lastModTime.Equal(info.ModTime()) && l.lastSize == info.Size()
	l.mutex.RUnlock()
	if unchanged {
		return
	}

	file, err := os.Open(path)
	if err != nil {
		l.setError(path, err.Error())
		return
	}
	defer file.Close()

	var entries []ipBanEntry
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		entry, ok := parseIPBanEntry(line)
		if !ok {
			logging.Warn("COMMON", "Ignoring invalid IP ban entry on line", lineNumber, "in", path, ":", line)
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		l.setError(path, err.Error())
		return
	}

	l.mutex.Lock()
	l.path = path
	l.lastModTime = info.ModTime()
	l.lastSize = info.Size()
	l.lastError = ""
	l.entries = entries
	l.mutex.Unlock()

	logging.Notice("COMMON", "Loaded", len(entries), "IP ban entries from", path)
}

func parseIPBanEntry(line string) (ipBanEntry, bool) {
	if strings.Contains(line, "/") {
		prefix, err := netip.ParsePrefix(line)
		if err != nil {
			return ipBanEntry{}, false
		}

		prefix = prefix.Masked()
		return ipBanEntry{
			label: line,
			from:  prefix.Addr().Unmap(),
			to:    lastAddrInPrefix(prefix),
		}, true
	}

	if strings.Contains(line, "-") {
		rangeParts := strings.SplitN(line, "-", 2)
		if len(rangeParts) != 2 {
			return ipBanEntry{}, false
		}

		start, err := netip.ParseAddr(strings.TrimSpace(rangeParts[0]))
		if err != nil {
			return ipBanEntry{}, false
		}

		end, err := netip.ParseAddr(strings.TrimSpace(rangeParts[1]))
		if err != nil {
			return ipBanEntry{}, false
		}

		start = start.Unmap()
		end = end.Unmap()
		if start.BitLen() != end.BitLen() || start.Compare(end) > 0 {
			return ipBanEntry{}, false
		}

		return ipBanEntry{
			label: line,
			from:  start,
			to:    end,
		}, true
	}

	addr, err := netip.ParseAddr(line)
	if err != nil {
		return ipBanEntry{}, false
	}

	addr = addr.Unmap()
	return ipBanEntry{
		label: line,
		from:  addr,
		to:    addr,
	}, true
}

func lastAddrInPrefix(prefix netip.Prefix) netip.Addr {
	addr := prefix.Addr().Unmap()
	bits := addr.BitLen()
	ones := prefix.Bits()

	bytes := addr.AsSlice()
	hostBits := bits - ones
	for i := len(bytes) - 1; i >= 0 && hostBits > 0; i-- {
		if hostBits >= 8 {
			bytes[i] = 0xFF
			hostBits -= 8
			continue
		}

		bytes[i] |= byte((1 << hostBits) - 1)
		hostBits = 0
	}

	last, ok := netip.AddrFromSlice(bytes)
	if !ok {
		return addr
	}

	return last.Unmap()
}

func (l *ipBanList) setError(path string, message string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.path == path && l.lastError == message {
		return
	}

	l.path = path
	l.lastModTime = time.Time{}
	l.lastSize = 0
	l.lastError = message
	l.entries = nil

	logging.Warn("COMMON", "Unable to load IP ban list from", path, ":", message)
}
