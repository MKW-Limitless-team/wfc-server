--
-- PostgreSQL database dump
--

-- Dumped from database version 14.9 (Ubuntu 14.9-0ubuntu0.22.04.1)
-- Dumped by pg_dump version 14.9 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: users; Type: TABLE; Schema: public; Owner: wiilink
--

CREATE TABLE IF NOT EXISTS public.users (
    profile_id bigint NOT NULL,
    user_id bigint NOT NULL,
    gsbrcd character varying NOT NULL,
    password character varying NOT NULL,
    ng_device_id bigint,
    email character varying NOT NULL,
    unique_nick character varying NOT NULL,
    firstname character varying,
    lastname character varying DEFAULT ''::character varying
);


ALTER TABLE ONLY public.users
    ADD IF NOT EXISTS last_ip_address character varying DEFAULT ''::character varying,
    ADD IF NOT EXISTS last_ingamesn character varying DEFAULT ''::character varying,
    ADD IF NOT EXISTS has_ban boolean DEFAULT false,
    ADD IF NOT EXISTS ban_issued timestamp without time zone,
    ADD IF NOT EXISTS ban_expires timestamp without time zone,
    ADD IF NOT EXISTS ban_reason character varying,
    ADD IF NOT EXISTS ban_reason_hidden character varying,
    ADD IF NOT EXISTS ban_moderator character varying,
    ADD IF NOT EXISTS ban_tos boolean,
	ADD IF NOT EXISTS open_host boolean DEFAULT false;

--
-- Change ng_device_id from bigint to bigint[]
--
DO $$ 
BEGIN
    IF (SELECT data_type FROM information_schema.columns WHERE table_name='users' AND column_name='ng_device_id') != 'ARRAY' THEN
        ALTER TABLE public.users
            ALTER COLUMN ng_device_id TYPE bigint[] using array[ng_device_id];
    END IF;
END $$;

ALTER TABLE public.users OWNER TO wiilink;

--
-- Name: sake_records; Type: TABLE; Schema: public; Owner: wiilink
--

CREATE TABLE IF NOT EXISTS public.sake_records (
    game_id integer NOT NULL,
    table_id character varying NOT NULL,
    record_id integer NOT NULL DEFAULT (random() * 2147483647)::integer,
    owner_id integer NOT NULL,
    fields jsonb NOT NULL CHECK (jsonb_typeof(fields) = 'object' AND jsonb_array_length(jsonb_path_query_array(fields, '$.keyvalue().key')) <= 64),
    create_time timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp without time zone DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT one_sake_record_constraint UNIQUE (game_id, table_id, record_id)
);

--
-- Name: mario_kart_wii_sake; Type: TABLE; Schema: public; Owner: wiilink
--

CREATE TABLE IF NOT EXISTS public.mario_kart_wii_sake (
    regionid smallint NOT NULL CHECK (regionid >= 1 AND regionid <= 7),
    courseid smallint NOT NULL CHECK (courseid >= 0 AND courseid <= 32767),
    score integer NOT NULL CHECK (score > 0 AND score < 360000),
    pid integer NOT NULL CHECK (pid > 0),
    playerinfo varchar(108) NOT NULL CHECK (LENGTH(playerinfo) = 108),
    ghost bytea CHECK (ghost IS NULL OR (OCTET_LENGTH(ghost) BETWEEN 148 AND 10240)),

    CONSTRAINT one_time_per_course_constraint UNIQUE (courseid, pid)
);


ALTER TABLE ONLY public.mario_kart_wii_sake
    ADD IF NOT EXISTS id serial PRIMARY KEY,
    ADD IF NOT EXISTS upload_time timestamp without time zone;


ALTER TABLE public.mario_kart_wii_sake OWNER TO wiilink;

--
-- Name: gamestats_public_data; Type: TABLE; Schema: public; Owner: wiilink
--

CREATE TABLE IF NOT EXISTS public.gamestats_public_data (
    profile_id bigint NOT NULL,
    dindex character varying NOT NULL,
    ptype character varying NOT NULL,
    pdata character varying NOT NULL,
    modified_time timestamp without time zone NOT NULL,

    CONSTRAINT one_pdata_constraint UNIQUE (profile_id, dindex, ptype)
);

--
-- Name: users_profile_id_seq; Type: SEQUENCE; Schema: public; Owner: wiilink
--

CREATE SEQUENCE IF NOT EXISTS public.users_profile_id_seq
    AS integer
    START WITH 1000000000
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.users_profile_id_seq OWNER TO wiilink;

--
-- Name: users_profile_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: wiilink
--

ALTER SEQUENCE public.users_profile_id_seq OWNED BY public.users.profile_id;

--
-- Name: users profile_id; Type: DEFAULT; Schema: public; Owner: wiilink
--

ALTER TABLE ONLY public.users ALTER COLUMN profile_id SET DEFAULT nextval('public.users_profile_id_seq'::regclass);

--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: wiilink
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (profile_id);

--
-- Name: tracks; Type: TABLE; Schema: public; Owner: wiilink
--

CREATE TABLE IF NOT EXISTS public.tracks (
    track_name character varying PRIMARY KEY,
    frequency bigint DEFAULT 0
);

--
-- Name: characters; Type: TABLE; Schema: public; Owner: wiilink
--

CREATE TABLE IF NOT EXISTS public.characters (
    profile_id bigint PRIMARY KEY,
    baby_daisy bigint DEFAULT 0,
    baby_luigi bigint DEFAULT 0,
    baby_mario bigint DEFAULT 0,
    baby_peach bigint DEFAULT 0,
    birdo bigint DEFAULT 0,
    bowser bigint DEFAULT 0,
    bowser_jr bigint DEFAULT 0,
    daisy bigint DEFAULT 0,
    diddy_kong bigint DEFAULT 0,
    donkey_kong bigint DEFAULT 0,
    dry_bones bigint DEFAULT 0,
    dry_bowser bigint DEFAULT 0,
    funky_kong bigint DEFAULT 0,
    king_boo bigint DEFAULT 0,
    koopa_troopa bigint DEFAULT 0,
    luigi bigint DEFAULT 0,
    mario bigint DEFAULT 0,
    mii_l_a_female bigint DEFAULT 0,
    mii_l_a_male bigint DEFAULT 0,
    mii_l_b_female bigint DEFAULT 0,
    mii_l_b_male bigint DEFAULT 0,
    mii_l_c_female bigint DEFAULT 0,
    mii_l_c_male bigint DEFAULT 0,
    mii_large bigint DEFAULT 0,
    mii_m_a_female bigint DEFAULT 0,
    mii_m_a_male bigint DEFAULT 0,
    mii_m_b_female bigint DEFAULT 0,
    mii_m_b_male bigint DEFAULT 0,
    mii_m_c_female bigint DEFAULT 0,
    mii_m_c_male bigint DEFAULT 0,
    mii_medium bigint DEFAULT 0,
    mii_s_a_female bigint DEFAULT 0,
    mii_s_a_male bigint DEFAULT 0,
    mii_s_b_female bigint DEFAULT 0,
    mii_s_b_male bigint DEFAULT 0,
    mii_s_c_female bigint DEFAULT 0,
    mii_s_c_male bigint DEFAULT 0,
    mii_small bigint DEFAULT 0,
    peach bigint DEFAULT 0,
    rosalina bigint DEFAULT 0,
    toad bigint DEFAULT 0,
    toadette bigint DEFAULT 0,
    wario bigint DEFAULT 0,
    waluigi bigint DEFAULT 0,
    yoshi bigint DEFAULT 0
);

--
-- Name: vehicles; Type: TABLE; Schema: public; Owner: wiilink
--

CREATE TABLE IF NOT EXISTS public.vehicles (
    profile_id bigint PRIMARY KEY,
    bit_bike bigint DEFAULT 0,
    blue_falcon bigint DEFAULT 0,
    booster_seat bigint DEFAULT 0,
    bullet_bike bigint DEFAULT 0,
    cheep_charger bigint DEFAULT 0,
    classic_dragster bigint DEFAULT 0,
    daytripper bigint DEFAULT 0,
    dolphin_dasher bigint DEFAULT 0,
    flame_flyer bigint DEFAULT 0,
    flame_runner bigint DEFAULT 0,
    honeycoupe bigint DEFAULT 0,
    jet_bubble bigint DEFAULT 0,
    jetsetter bigint DEFAULT 0,
    mach_bike bigint DEFAULT 0,
    magikruiser bigint DEFAULT 0,
    mini_beast bigint DEFAULT 0,
    offroader bigint DEFAULT 0,
    phantom bigint DEFAULT 0,
    piranha_prowler bigint DEFAULT 0,
    quacker bigint DEFAULT 0,
    shooting_star bigint DEFAULT 0,
    sneakster bigint DEFAULT 0,
    spear bigint DEFAULT 0,
    sprinter bigint DEFAULT 0,
    standard_bike_l bigint DEFAULT 0,
    standard_bike_m bigint DEFAULT 0,
    standard_bike_s bigint DEFAULT 0,
    standard_kart_l bigint DEFAULT 0,
    standard_kart_m bigint DEFAULT 0,
    standard_kart_s bigint DEFAULT 0,
    sugarscoot bigint DEFAULT 0,
    super_blooper bigint DEFAULT 0,
    tiny_titan bigint DEFAULT 0,
    wario_bike bigint DEFAULT 0,
    wild_wing bigint DEFAULT 0,
    zip_zip bigint DEFAULT 0

);

--
-- Name: events; Type: TABLE; Schema: public; Owner: wiilink
--

CREATE TABLE IF NOT EXISTS public.events (
    id serial PRIMARY KEY,
    event_type character varying NOT NULL,
    event_data jsonb NOT NULL,
    event_time timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);

--
-- PostgreSQL database dump complete
--
