--
-- PostgreSQL database dump
--

-- Dumped from database version 14.5 (Debian 14.5-1.pgdg110+1)
-- Dumped by pg_dump version 14.3

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

--
-- Name: create_partition(timestamp with time zone, timestamp with time zone); Type: FUNCTION; Schema: public; Owner: crc
--

CREATE FUNCTION public.create_partition(start timestamp with time zone, stop timestamp with time zone) RETURNS void
    LANGUAGE plpgsql
    AS $$
            DECLARE
                start TEXT := get_date_string(START);
                stop TEXT := get_date_string(STOP);
                partition VARCHAR := FORMAT('partition_%s_%s', start, stop);
            BEGIN
                EXECUTE 'CREATE TABLE IF NOT EXISTS ' || partition || ' PARTITION OF payload_statuses FOR VALUES FROM (' || quote_literal(START) || ') TO (' || quote_literal(STOP) || ');';
                EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS ' || partition || '_id_idx ON ' || partition || ' USING btree(id);';
                EXECUTE 'CREATE INDEX IF NOT EXISTS ' || partition || '_payload_id_idx ON ' || partition || ' USING btree(payload_id);';
                EXECUTE 'CREATE INDEX IF NOT EXISTS ' || partition || '_service_id_idx ON ' || partition || ' USING btree(service_id);';
                EXECUTE 'CREATE INDEX IF NOT EXISTS ' || partition || '_source_id_idx ON ' || partition || ' USING btree(source_id);';
                EXECUTE 'CREATE INDEX IF NOT EXISTS ' || partition || '_status_id_idx ON ' || partition || ' USING btree(status_id);';
                EXECUTE 'CREATE INDEX IF NOT EXISTS ' || partition || '_date_idx ON ' || partition || ' USING btree(date);';
                EXECUTE 'CREATE INDEX IF NOT EXISTS ' || partition || '_created_at_idx ON ' || partition || ' USING btree(created_at);';
            END;
        $$;


ALTER FUNCTION public.create_partition(start timestamp with time zone, stop timestamp with time zone) OWNER TO crc;

--
-- Name: drop_partition(timestamp with time zone, timestamp with time zone); Type: FUNCTION; Schema: public; Owner: crc
--

CREATE FUNCTION public.drop_partition(start timestamp with time zone, stop timestamp with time zone) RETURNS void
    LANGUAGE plpgsql
    AS $$
            DECLARE
                start TEXT := get_date_string(START);
                stop TEXT := get_date_string(STOP);
                partition VARCHAR := FORMAT('partition_%s_%s', start, stop);
            BEGIN
                EXECUTE 'DROP TABLE IF EXISTS ' || partition || ';';
            END;
        $$;


ALTER FUNCTION public.drop_partition(start timestamp with time zone, stop timestamp with time zone) OWNER TO crc;

--
-- Name: get_date_string(timestamp with time zone); Type: FUNCTION; Schema: public; Owner: crc
--

CREATE FUNCTION public.get_date_string(value timestamp with time zone) RETURNS text
    LANGUAGE plpgsql
    AS $$
            BEGIN
                RETURN CAST((
                    EXTRACT(DAY from VALUE
                ) + (100 * EXTRACT(
                    MONTH from VALUE
                )) + (10000 * EXTRACT(
                    YEAR from VALUE))) AS TEXT);
            END
        $$;


ALTER FUNCTION public.get_date_string(value timestamp with time zone) OWNER TO crc;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: alembic_version; Type: TABLE; Schema: public; Owner: crc
--

CREATE TABLE public.alembic_version (
    version_num character varying(32) NOT NULL
);


ALTER TABLE public.alembic_version OWNER TO crc;

--
-- Name: payload_statuses; Type: TABLE; Schema: public; Owner: crc
--

CREATE TABLE public.payload_statuses (
    id bigint NOT NULL,
    payload_id bigint NOT NULL,
    service_id integer NOT NULL,
    source_id integer,
    status_id integer NOT NULL,
    status_msg character varying,
    date timestamp with time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    created_at timestamp with time zone DEFAULT timezone('utc'::text, now()) NOT NULL
)
PARTITION BY RANGE (date);


ALTER TABLE public.payload_statuses OWNER TO crc;

--
-- Name: payload_statuses_id_seq; Type: SEQUENCE; Schema: public; Owner: crc
--

CREATE SEQUENCE public.payload_statuses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.payload_statuses_id_seq OWNER TO crc;

--
-- Name: payload_statuses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: crc
--

ALTER SEQUENCE public.payload_statuses_id_seq OWNED BY public.payload_statuses.id;


--
-- Name: partition_20240212_20240213; Type: TABLE; Schema: public; Owner: crc
--

CREATE TABLE public.partition_20240212_20240213 (
    id bigint DEFAULT nextval('public.payload_statuses_id_seq'::regclass) NOT NULL,
    payload_id bigint NOT NULL,
    service_id integer NOT NULL,
    source_id integer,
    status_id integer NOT NULL,
    status_msg character varying,
    date timestamp with time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    created_at timestamp with time zone DEFAULT timezone('utc'::text, now()) NOT NULL
);


ALTER TABLE public.partition_20240212_20240213 OWNER TO crc;

--
-- Name: partition_20240213_20240214; Type: TABLE; Schema: public; Owner: crc
--

CREATE TABLE public.partition_20240213_20240214 (
    id bigint DEFAULT nextval('public.payload_statuses_id_seq'::regclass) NOT NULL,
    payload_id bigint NOT NULL,
    service_id integer NOT NULL,
    source_id integer,
    status_id integer NOT NULL,
    status_msg character varying,
    date timestamp with time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    created_at timestamp with time zone DEFAULT timezone('utc'::text, now()) NOT NULL
);


ALTER TABLE public.partition_20240213_20240214 OWNER TO crc;

--
-- Name: payloads; Type: TABLE; Schema: public; Owner: crc
--

CREATE TABLE public.payloads (
    id bigint NOT NULL,
    request_id character varying NOT NULL,
    account character varying,
    inventory_id character varying,
    system_id character varying,
    created_at timestamp with time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    org_id character varying
);


ALTER TABLE public.payloads OWNER TO crc;

--
-- Name: payloads_id_seq; Type: SEQUENCE; Schema: public; Owner: crc
--

CREATE SEQUENCE public.payloads_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.payloads_id_seq OWNER TO crc;

--
-- Name: payloads_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: crc
--

ALTER SEQUENCE public.payloads_id_seq OWNED BY public.payloads.id;


--
-- Name: services; Type: TABLE; Schema: public; Owner: crc
--

CREATE TABLE public.services (
    id integer NOT NULL,
    name character varying NOT NULL
);


ALTER TABLE public.services OWNER TO crc;

--
-- Name: services_id_seq; Type: SEQUENCE; Schema: public; Owner: crc
--

CREATE SEQUENCE public.services_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.services_id_seq OWNER TO crc;

--
-- Name: services_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: crc
--

ALTER SEQUENCE public.services_id_seq OWNED BY public.services.id;


--
-- Name: sources; Type: TABLE; Schema: public; Owner: crc
--

CREATE TABLE public.sources (
    id integer NOT NULL,
    name character varying NOT NULL
);


ALTER TABLE public.sources OWNER TO crc;

--
-- Name: sources_id_seq; Type: SEQUENCE; Schema: public; Owner: crc
--

CREATE SEQUENCE public.sources_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.sources_id_seq OWNER TO crc;

--
-- Name: sources_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: crc
--

ALTER SEQUENCE public.sources_id_seq OWNED BY public.sources.id;


--
-- Name: statuses; Type: TABLE; Schema: public; Owner: crc
--

CREATE TABLE public.statuses (
    id integer NOT NULL,
    name character varying NOT NULL
);


ALTER TABLE public.statuses OWNER TO crc;

--
-- Name: statuses_id_seq; Type: SEQUENCE; Schema: public; Owner: crc
--

CREATE SEQUENCE public.statuses_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.statuses_id_seq OWNER TO crc;

--
-- Name: statuses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: crc
--

ALTER SEQUENCE public.statuses_id_seq OWNED BY public.statuses.id;


--
-- Name: partition_20240212_20240213; Type: TABLE ATTACH; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.payload_statuses ATTACH PARTITION public.partition_20240212_20240213 FOR VALUES FROM ('2024-02-12 00:00:00+00') TO ('2024-02-13 00:00:00+00');


--
-- Name: partition_20240213_20240214; Type: TABLE ATTACH; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.payload_statuses ATTACH PARTITION public.partition_20240213_20240214 FOR VALUES FROM ('2024-02-13 00:00:00+00') TO ('2024-02-14 00:00:00+00');


--
-- Name: payload_statuses id; Type: DEFAULT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.payload_statuses ALTER COLUMN id SET DEFAULT nextval('public.payload_statuses_id_seq'::regclass);


--
-- Name: payloads id; Type: DEFAULT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.payloads ALTER COLUMN id SET DEFAULT nextval('public.payloads_id_seq'::regclass);


--
-- Name: services id; Type: DEFAULT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.services ALTER COLUMN id SET DEFAULT nextval('public.services_id_seq'::regclass);


--
-- Name: sources id; Type: DEFAULT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.sources ALTER COLUMN id SET DEFAULT nextval('public.sources_id_seq'::regclass);


--
-- Name: statuses id; Type: DEFAULT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.statuses ALTER COLUMN id SET DEFAULT nextval('public.statuses_id_seq'::regclass);


--
-- Data for Name: alembic_version; Type: TABLE DATA; Schema: public; Owner: crc
--

COPY public.alembic_version (version_num) FROM stdin;
4cd848af7d8c
\.


--
-- Data for Name: partition_20240212_20240213; Type: TABLE DATA; Schema: public; Owner: crc
--

COPY public.partition_20240212_20240213 (id, payload_id, service_id, source_id, status_id, status_msg, date, created_at) FROM stdin;
\.


--
-- Data for Name: partition_20240213_20240214; Type: TABLE DATA; Schema: public; Owner: crc
--

COPY public.partition_20240213_20240214 (id, payload_id, service_id, source_id, status_id, status_msg, date, created_at) FROM stdin;
\.


--
-- Data for Name: payloads; Type: TABLE DATA; Schema: public; Owner: crc
--

COPY public.payloads (id, request_id, account, inventory_id, system_id, created_at, org_id) FROM stdin;
\.


--
-- Data for Name: services; Type: TABLE DATA; Schema: public; Owner: crc
--

COPY public.services (id, name) FROM stdin;
1	advisor
2	ccx-data-pipeline
3	compliance
4	hsp-archiver
5	hsp-deleter
6	ingress
7	insights-advisor-service
8	insights-engine
9	insights-results-db-writer
10	inventory
11	inventory-mq-service
12	puptoo
13	vulnerability
\.


--
-- Data for Name: sources; Type: TABLE DATA; Schema: public; Owner: crc
--

COPY public.sources (id, name) FROM stdin;
1	compliance-consumer
2	compliance-sidekiq
3	insights-client
4	inventory
\.


--
-- Data for Name: statuses; Type: TABLE DATA; Schema: public; Owner: crc
--

COPY public.statuses (id, name) FROM stdin;
1	error
2	failed
3	processed
4	processing
5	processing_error
6	processing_success
7	recieved
8	success
\.


--
-- Name: payload_statuses_id_seq; Type: SEQUENCE SET; Schema: public; Owner: crc
--

SELECT pg_catalog.setval('public.payload_statuses_id_seq', 1, false);


--
-- Name: payloads_id_seq; Type: SEQUENCE SET; Schema: public; Owner: crc
--

SELECT pg_catalog.setval('public.payloads_id_seq', 1, false);


--
-- Name: services_id_seq; Type: SEQUENCE SET; Schema: public; Owner: crc
--

SELECT pg_catalog.setval('public.services_id_seq', 13, true);


--
-- Name: sources_id_seq; Type: SEQUENCE SET; Schema: public; Owner: crc
--

SELECT pg_catalog.setval('public.sources_id_seq', 4, true);


--
-- Name: statuses_id_seq; Type: SEQUENCE SET; Schema: public; Owner: crc
--

SELECT pg_catalog.setval('public.statuses_id_seq', 8, true);


--
-- Name: alembic_version alembic_version_pkc; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.alembic_version
    ADD CONSTRAINT alembic_version_pkc PRIMARY KEY (version_num);


--
-- Name: services idx_services_name; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.services
    ADD CONSTRAINT idx_services_name UNIQUE (name);


--
-- Name: sources idx_sources_name; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.sources
    ADD CONSTRAINT idx_sources_name UNIQUE (name);


--
-- Name: statuses idx_statuses_name; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.statuses
    ADD CONSTRAINT idx_statuses_name UNIQUE (name);


--
-- Name: payload_statuses payload_statuses_pkey; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.payload_statuses
    ADD CONSTRAINT payload_statuses_pkey PRIMARY KEY (id, date);


--
-- Name: partition_20240212_20240213 partition_20240212_20240213_pkey; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.partition_20240212_20240213
    ADD CONSTRAINT partition_20240212_20240213_pkey PRIMARY KEY (id, date);


--
-- Name: partition_20240213_20240214 partition_20240213_20240214_pkey; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.partition_20240213_20240214
    ADD CONSTRAINT partition_20240213_20240214_pkey PRIMARY KEY (id, date);


--
-- Name: payloads payloads_pkey; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.payloads
    ADD CONSTRAINT payloads_pkey PRIMARY KEY (id);


--
-- Name: payloads payloads_request_id_key; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.payloads
    ADD CONSTRAINT payloads_request_id_key UNIQUE (request_id);


--
-- Name: services services_name_key; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.services
    ADD CONSTRAINT services_name_key UNIQUE (name);


--
-- Name: services services_pkey; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.services
    ADD CONSTRAINT services_pkey PRIMARY KEY (id);


--
-- Name: sources sources_name_key; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.sources
    ADD CONSTRAINT sources_name_key UNIQUE (name);


--
-- Name: sources sources_pkey; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.sources
    ADD CONSTRAINT sources_pkey PRIMARY KEY (id);


--
-- Name: statuses statuses_name_key; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.statuses
    ADD CONSTRAINT statuses_name_key UNIQUE (name);


--
-- Name: statuses statuses_pkey; Type: CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE ONLY public.statuses
    ADD CONSTRAINT statuses_pkey PRIMARY KEY (id);


--
-- Name: partition_20240212_20240213_created_at_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_created_at_idx ON public.partition_20240212_20240213 USING btree (created_at);


--
-- Name: partition_20240212_20240213_date_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_date_idx ON public.partition_20240212_20240213 USING btree (date);


--
-- Name: partition_20240212_20240213_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE UNIQUE INDEX partition_20240212_20240213_id_idx ON public.partition_20240212_20240213 USING btree (id);


--
-- Name: partition_20240212_20240213_payload_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_payload_id_idx ON public.partition_20240212_20240213 USING btree (payload_id);


--
-- Name: payload_statuses_service_id_created_at_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX payload_statuses_service_id_created_at_idx ON ONLY public.payload_statuses USING btree (service_id, created_at);


--
-- Name: partition_20240212_20240213_service_id_created_at_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_service_id_created_at_idx ON public.partition_20240212_20240213 USING btree (service_id, created_at);


--
-- Name: payload_statuses_service_id_date_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX payload_statuses_service_id_date_idx ON ONLY public.payload_statuses USING btree (service_id, date);


--
-- Name: partition_20240212_20240213_service_id_date_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_service_id_date_idx ON public.partition_20240212_20240213 USING btree (service_id, date);


--
-- Name: partition_20240212_20240213_service_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_service_id_idx ON public.partition_20240212_20240213 USING btree (service_id);


--
-- Name: payload_statuses_service_id_status_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX payload_statuses_service_id_status_id_idx ON ONLY public.payload_statuses USING btree (service_id, status_id);


--
-- Name: partition_20240212_20240213_service_id_status_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_service_id_status_id_idx ON public.partition_20240212_20240213 USING btree (service_id, status_id);


--
-- Name: partition_20240212_20240213_source_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_source_id_idx ON public.partition_20240212_20240213 USING btree (source_id);


--
-- Name: payload_statuses_status_id_created_at_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX payload_statuses_status_id_created_at_idx ON ONLY public.payload_statuses USING btree (status_id, created_at);


--
-- Name: partition_20240212_20240213_status_id_created_at_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_status_id_created_at_idx ON public.partition_20240212_20240213 USING btree (status_id, created_at);


--
-- Name: payload_statuses_source_id_created_at_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX payload_statuses_source_id_created_at_idx ON ONLY public.payload_statuses USING btree (status_id, created_at);


--
-- Name: partition_20240212_20240213_status_id_created_at_idx1; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_status_id_created_at_idx1 ON public.partition_20240212_20240213 USING btree (status_id, created_at);


--
-- Name: payload_statuses_status_id_date_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX payload_statuses_status_id_date_idx ON ONLY public.payload_statuses USING btree (status_id, date);


--
-- Name: partition_20240212_20240213_status_id_date_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_status_id_date_idx ON public.partition_20240212_20240213 USING btree (status_id, date);


--
-- Name: payload_statuses_source_id_date_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX payload_statuses_source_id_date_idx ON ONLY public.payload_statuses USING btree (status_id, date);


--
-- Name: partition_20240212_20240213_status_id_date_idx1; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_status_id_date_idx1 ON public.partition_20240212_20240213 USING btree (status_id, date);


--
-- Name: partition_20240212_20240213_status_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240212_20240213_status_id_idx ON public.partition_20240212_20240213 USING btree (status_id);


--
-- Name: partition_20240213_20240214_created_at_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_created_at_idx ON public.partition_20240213_20240214 USING btree (created_at);


--
-- Name: partition_20240213_20240214_date_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_date_idx ON public.partition_20240213_20240214 USING btree (date);


--
-- Name: partition_20240213_20240214_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE UNIQUE INDEX partition_20240213_20240214_id_idx ON public.partition_20240213_20240214 USING btree (id);


--
-- Name: partition_20240213_20240214_payload_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_payload_id_idx ON public.partition_20240213_20240214 USING btree (payload_id);


--
-- Name: partition_20240213_20240214_service_id_created_at_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_service_id_created_at_idx ON public.partition_20240213_20240214 USING btree (service_id, created_at);


--
-- Name: partition_20240213_20240214_service_id_date_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_service_id_date_idx ON public.partition_20240213_20240214 USING btree (service_id, date);


--
-- Name: partition_20240213_20240214_service_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_service_id_idx ON public.partition_20240213_20240214 USING btree (service_id);


--
-- Name: partition_20240213_20240214_service_id_status_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_service_id_status_id_idx ON public.partition_20240213_20240214 USING btree (service_id, status_id);


--
-- Name: partition_20240213_20240214_source_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_source_id_idx ON public.partition_20240213_20240214 USING btree (source_id);


--
-- Name: partition_20240213_20240214_status_id_created_at_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_status_id_created_at_idx ON public.partition_20240213_20240214 USING btree (status_id, created_at);


--
-- Name: partition_20240213_20240214_status_id_created_at_idx1; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_status_id_created_at_idx1 ON public.partition_20240213_20240214 USING btree (status_id, created_at);


--
-- Name: partition_20240213_20240214_status_id_date_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_status_id_date_idx ON public.partition_20240213_20240214 USING btree (status_id, date);


--
-- Name: partition_20240213_20240214_status_id_date_idx1; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_status_id_date_idx1 ON public.partition_20240213_20240214 USING btree (status_id, date);


--
-- Name: partition_20240213_20240214_status_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX partition_20240213_20240214_status_id_idx ON public.partition_20240213_20240214 USING btree (status_id);


--
-- Name: payloads_account_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX payloads_account_idx ON public.payloads USING btree (account);


--
-- Name: payloads_created_at_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX payloads_created_at_idx ON public.payloads USING btree (created_at);


--
-- Name: payloads_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE UNIQUE INDEX payloads_id_idx ON public.payloads USING btree (id);


--
-- Name: payloads_inventory_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX payloads_inventory_id_idx ON public.payloads USING btree (inventory_id);


--
-- Name: payloads_request_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE UNIQUE INDEX payloads_request_id_idx ON public.payloads USING btree (request_id);


--
-- Name: payloads_system_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE INDEX payloads_system_id_idx ON public.payloads USING btree (system_id);


--
-- Name: services_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE UNIQUE INDEX services_id_idx ON public.services USING btree (id);


--
-- Name: services_name_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE UNIQUE INDEX services_name_idx ON public.services USING btree (name);


--
-- Name: sources_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE UNIQUE INDEX sources_id_idx ON public.sources USING btree (id);


--
-- Name: sources_name_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE UNIQUE INDEX sources_name_idx ON public.sources USING btree (name);


--
-- Name: statuses_id_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE UNIQUE INDEX statuses_id_idx ON public.statuses USING btree (id);


--
-- Name: statuses_name_idx; Type: INDEX; Schema: public; Owner: crc
--

CREATE UNIQUE INDEX statuses_name_idx ON public.statuses USING btree (name);


--
-- Name: partition_20240212_20240213_pkey; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_pkey ATTACH PARTITION public.partition_20240212_20240213_pkey;


--
-- Name: partition_20240212_20240213_service_id_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_service_id_created_at_idx ATTACH PARTITION public.partition_20240212_20240213_service_id_created_at_idx;


--
-- Name: partition_20240212_20240213_service_id_date_idx; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_service_id_date_idx ATTACH PARTITION public.partition_20240212_20240213_service_id_date_idx;


--
-- Name: partition_20240212_20240213_service_id_status_id_idx; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_service_id_status_id_idx ATTACH PARTITION public.partition_20240212_20240213_service_id_status_id_idx;


--
-- Name: partition_20240212_20240213_status_id_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_status_id_created_at_idx ATTACH PARTITION public.partition_20240212_20240213_status_id_created_at_idx;


--
-- Name: partition_20240212_20240213_status_id_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_source_id_created_at_idx ATTACH PARTITION public.partition_20240212_20240213_status_id_created_at_idx1;


--
-- Name: partition_20240212_20240213_status_id_date_idx; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_status_id_date_idx ATTACH PARTITION public.partition_20240212_20240213_status_id_date_idx;


--
-- Name: partition_20240212_20240213_status_id_date_idx1; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_source_id_date_idx ATTACH PARTITION public.partition_20240212_20240213_status_id_date_idx1;


--
-- Name: partition_20240213_20240214_pkey; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_pkey ATTACH PARTITION public.partition_20240213_20240214_pkey;


--
-- Name: partition_20240213_20240214_service_id_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_service_id_created_at_idx ATTACH PARTITION public.partition_20240213_20240214_service_id_created_at_idx;


--
-- Name: partition_20240213_20240214_service_id_date_idx; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_service_id_date_idx ATTACH PARTITION public.partition_20240213_20240214_service_id_date_idx;


--
-- Name: partition_20240213_20240214_service_id_status_id_idx; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_service_id_status_id_idx ATTACH PARTITION public.partition_20240213_20240214_service_id_status_id_idx;


--
-- Name: partition_20240213_20240214_status_id_created_at_idx; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_status_id_created_at_idx ATTACH PARTITION public.partition_20240213_20240214_status_id_created_at_idx;


--
-- Name: partition_20240213_20240214_status_id_created_at_idx1; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_source_id_created_at_idx ATTACH PARTITION public.partition_20240213_20240214_status_id_created_at_idx1;


--
-- Name: partition_20240213_20240214_status_id_date_idx; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_status_id_date_idx ATTACH PARTITION public.partition_20240213_20240214_status_id_date_idx;


--
-- Name: partition_20240213_20240214_status_id_date_idx1; Type: INDEX ATTACH; Schema: public; Owner: crc
--

ALTER INDEX public.payload_statuses_source_id_date_idx ATTACH PARTITION public.partition_20240213_20240214_status_id_date_idx1;


--
-- Name: payload_statuses fk_payload_statuses_payload; Type: FK CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE public.payload_statuses
    ADD CONSTRAINT fk_payload_statuses_payload FOREIGN KEY (payload_id) REFERENCES public.payloads(id);


--
-- Name: payload_statuses fk_payload_statuses_service; Type: FK CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE public.payload_statuses
    ADD CONSTRAINT fk_payload_statuses_service FOREIGN KEY (service_id) REFERENCES public.services(id);


--
-- Name: payload_statuses fk_payload_statuses_source; Type: FK CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE public.payload_statuses
    ADD CONSTRAINT fk_payload_statuses_source FOREIGN KEY (source_id) REFERENCES public.sources(id);


--
-- Name: payload_statuses fk_payload_statuses_status; Type: FK CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE public.payload_statuses
    ADD CONSTRAINT fk_payload_statuses_status FOREIGN KEY (status_id) REFERENCES public.statuses(id);


--
-- Name: payload_statuses payload_statuses_payload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE public.payload_statuses
    ADD CONSTRAINT payload_statuses_payload_id_fkey FOREIGN KEY (payload_id) REFERENCES public.payloads(id) ON DELETE CASCADE;


--
-- Name: payload_statuses payload_statuses_service_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE public.payload_statuses
    ADD CONSTRAINT payload_statuses_service_id_fkey FOREIGN KEY (service_id) REFERENCES public.services(id);


--
-- Name: payload_statuses payload_statuses_source_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE public.payload_statuses
    ADD CONSTRAINT payload_statuses_source_id_fkey FOREIGN KEY (source_id) REFERENCES public.sources(id);


--
-- Name: payload_statuses payload_statuses_status_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: crc
--

ALTER TABLE public.payload_statuses
    ADD CONSTRAINT payload_statuses_status_id_fkey FOREIGN KEY (status_id) REFERENCES public.statuses(id);


--
-- PostgreSQL database dump complete
--

