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


CREATE SEQUENCE public.payload_statuses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.payload_statuses_id_seq OWNED BY public.payload_statuses.id;


CREATE TABLE public.payloads (
    id bigint NOT NULL,
    request_id character varying NOT NULL,
    account character varying,
    inventory_id character varying,
    system_id character varying,
    created_at timestamp with time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    org_id character varying
);


CREATE SEQUENCE public.payloads_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.payloads_id_seq OWNED BY public.payloads.id;


CREATE TABLE public.services (
    id integer NOT NULL,
    name character varying NOT NULL
);


CREATE SEQUENCE public.services_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.services_id_seq OWNED BY public.services.id;


CREATE TABLE public.sources (
    id integer NOT NULL,
    name character varying NOT NULL
);

CREATE SEQUENCE public.sources_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.sources_id_seq OWNED BY public.sources.id;


CREATE TABLE public.statuses (
    id integer NOT NULL,
    name character varying NOT NULL
);


CREATE SEQUENCE public.statuses_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.statuses_id_seq OWNED BY public.statuses.id;


ALTER TABLE ONLY public.payload_statuses ALTER COLUMN id SET DEFAULT nextval('public.payload_statuses_id_seq'::regclass);

ALTER TABLE ONLY public.payloads ALTER COLUMN id SET DEFAULT nextval('public.payloads_id_seq'::regclass);

ALTER TABLE ONLY public.services ALTER COLUMN id SET DEFAULT nextval('public.services_id_seq'::regclass);

ALTER TABLE ONLY public.sources ALTER COLUMN id SET DEFAULT nextval('public.sources_id_seq'::regclass);

ALTER TABLE ONLY public.statuses ALTER COLUMN id SET DEFAULT nextval('public.statuses_id_seq'::regclass);


SELECT pg_catalog.setval('public.payload_statuses_id_seq', 1, false);

SELECT pg_catalog.setval('public.payloads_id_seq', 1, false);

SELECT pg_catalog.setval('public.services_id_seq', 13, true);

SELECT pg_catalog.setval('public.sources_id_seq', 4, true);

SELECT pg_catalog.setval('public.statuses_id_seq', 8, true);


ALTER TABLE ONLY public.services
    ADD CONSTRAINT idx_services_name UNIQUE (name);

ALTER TABLE ONLY public.sources
    ADD CONSTRAINT idx_sources_name UNIQUE (name);

ALTER TABLE ONLY public.statuses
    ADD CONSTRAINT idx_statuses_name UNIQUE (name);

ALTER TABLE ONLY public.payload_statuses
    ADD CONSTRAINT payload_statuses_pkey PRIMARY KEY (id, date);

ALTER TABLE ONLY public.payloads
    ADD CONSTRAINT payloads_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.payloads
    ADD CONSTRAINT payloads_request_id_key UNIQUE (request_id);

ALTER TABLE ONLY public.services
    ADD CONSTRAINT services_name_key UNIQUE (name);

ALTER TABLE ONLY public.services
    ADD CONSTRAINT services_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.sources
    ADD CONSTRAINT sources_name_key UNIQUE (name);

ALTER TABLE ONLY public.sources
    ADD CONSTRAINT sources_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.statuses
    ADD CONSTRAINT statuses_name_key UNIQUE (name);

ALTER TABLE ONLY public.statuses
    ADD CONSTRAINT statuses_pkey PRIMARY KEY (id);



CREATE INDEX payload_statuses_service_id_created_at_idx ON ONLY public.payload_statuses USING btree (service_id, created_at);

CREATE INDEX payload_statuses_service_id_date_idx ON ONLY public.payload_statuses USING btree (service_id, date);

CREATE INDEX payload_statuses_service_id_status_id_idx ON ONLY public.payload_statuses USING btree (service_id, status_id);

CREATE INDEX payload_statuses_status_id_created_at_idx ON ONLY public.payload_statuses USING btree (status_id, created_at);

CREATE INDEX payload_statuses_source_id_created_at_idx ON ONLY public.payload_statuses USING btree (status_id, created_at);

CREATE INDEX payload_statuses_status_id_date_idx ON ONLY public.payload_statuses USING btree (status_id, date);

CREATE INDEX payload_statuses_source_id_date_idx ON ONLY public.payload_statuses USING btree (status_id, date);

CREATE INDEX payloads_account_idx ON public.payloads USING btree (account);

CREATE INDEX payloads_created_at_idx ON payloads USING btree (created_at);

CREATE UNIQUE INDEX payloads_id_idx ON public.payloads USING btree (id);

CREATE INDEX payloads_inventory_id_idx ON public.payloads USING btree (inventory_id);

CREATE UNIQUE INDEX payloads_request_id_idx ON public.payloads USING btree (request_id);

CREATE INDEX payloads_system_id_idx ON public.payloads USING btree (system_id);

CREATE UNIQUE INDEX services_id_idx ON public.services USING btree (id);

CREATE UNIQUE INDEX services_name_idx ON public.services USING btree (name);

CREATE UNIQUE INDEX sources_id_idx ON public.sources USING btree (id);

CREATE UNIQUE INDEX sources_name_idx ON public.sources USING btree (name);

CREATE UNIQUE INDEX statuses_id_idx ON public.statuses USING btree (id);

CREATE UNIQUE INDEX statuses_name_idx ON public.statuses USING btree (name);

ALTER TABLE public.payload_statuses
    ADD CONSTRAINT payload_statuses_payload_id_fkey FOREIGN KEY (payload_id) REFERENCES public.payloads(id) ON DELETE CASCADE;

ALTER TABLE public.payload_statuses
    ADD CONSTRAINT payload_statuses_service_id_fkey FOREIGN KEY (service_id) REFERENCES public.services(id);

ALTER TABLE public.payload_statuses
    ADD CONSTRAINT payload_statuses_source_id_fkey FOREIGN KEY (source_id) REFERENCES public.sources(id);

ALTER TABLE public.payload_statuses
    ADD CONSTRAINT payload_statuses_status_id_fkey FOREIGN KEY (status_id) REFERENCES public.statuses(id);


SELECT create_partition(NOW()::DATE, NOW()::DATE + interval '1 DAY');
