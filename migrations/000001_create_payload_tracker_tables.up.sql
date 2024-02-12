CREATE FUNCTION create_partition(start timestamp with time zone, stop timestamp with time zone) RETURNS void
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


CREATE FUNCTION drop_partition(start timestamp with time zone, stop timestamp with time zone) RETURNS void
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


CREATE FUNCTION get_date_string(value timestamp with time zone) RETURNS text
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



CREATE TABLE payload_statuses (
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


CREATE SEQUENCE payload_statuses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE payload_statuses_id_seq OWNED BY payload_statuses.id;


CREATE TABLE payloads (
    id bigint NOT NULL,
    request_id character varying NOT NULL,
    account character varying,
    inventory_id character varying,
    system_id character varying,
    created_at timestamp with time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    org_id character varying
);


CREATE SEQUENCE payloads_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE payloads_id_seq OWNED BY payloads.id;


CREATE TABLE services (
    id integer NOT NULL,
    name character varying NOT NULL
);


CREATE SEQUENCE services_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE services_id_seq OWNED BY services.id;


CREATE TABLE sources (
    id integer NOT NULL,
    name character varying NOT NULL
);

CREATE SEQUENCE sources_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE sources_id_seq OWNED BY sources.id;


CREATE TABLE statuses (
    id integer NOT NULL,
    name character varying NOT NULL
);


CREATE SEQUENCE statuses_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE statuses_id_seq OWNED BY statuses.id;


ALTER TABLE ONLY payload_statuses ALTER COLUMN id SET DEFAULT nextval('payload_statuses_id_seq'::regclass);

ALTER TABLE ONLY payloads ALTER COLUMN id SET DEFAULT nextval('payloads_id_seq'::regclass);

ALTER TABLE ONLY services ALTER COLUMN id SET DEFAULT nextval('services_id_seq'::regclass);

ALTER TABLE ONLY sources ALTER COLUMN id SET DEFAULT nextval('sources_id_seq'::regclass);

ALTER TABLE ONLY statuses ALTER COLUMN id SET DEFAULT nextval('statuses_id_seq'::regclass);


SELECT pg_catalog.setval('payload_statuses_id_seq', 1, false);

SELECT pg_catalog.setval('payloads_id_seq', 1, false);

SELECT pg_catalog.setval('services_id_seq', 13, true);

SELECT pg_catalog.setval('sources_id_seq', 4, true);

SELECT pg_catalog.setval('statuses_id_seq', 8, true);


ALTER TABLE ONLY services
    ADD CONSTRAINT idx_services_name UNIQUE (name);

ALTER TABLE ONLY sources
    ADD CONSTRAINT idx_sources_name UNIQUE (name);

ALTER TABLE ONLY statuses
    ADD CONSTRAINT idx_statuses_name UNIQUE (name);

ALTER TABLE ONLY payload_statuses
    ADD CONSTRAINT payload_statuses_pkey PRIMARY KEY (id, date);

ALTER TABLE ONLY payloads
    ADD CONSTRAINT payloads_pkey PRIMARY KEY (id);

ALTER TABLE ONLY payloads
    ADD CONSTRAINT payloads_request_id_key UNIQUE (request_id);

ALTER TABLE ONLY services
    ADD CONSTRAINT services_name_key UNIQUE (name);

ALTER TABLE ONLY services
    ADD CONSTRAINT services_pkey PRIMARY KEY (id);

ALTER TABLE ONLY sources
    ADD CONSTRAINT sources_name_key UNIQUE (name);

ALTER TABLE ONLY sources
    ADD CONSTRAINT sources_pkey PRIMARY KEY (id);

ALTER TABLE ONLY statuses
    ADD CONSTRAINT statuses_name_key UNIQUE (name);

ALTER TABLE ONLY statuses
    ADD CONSTRAINT statuses_pkey PRIMARY KEY (id);


CREATE INDEX payload_statuses_service_id_created_at_idx ON ONLY payload_statuses USING btree (service_id, created_at);

CREATE INDEX payload_statuses_service_id_date_idx ON ONLY payload_statuses USING btree (service_id, date);

CREATE INDEX payload_statuses_service_id_status_id_idx ON ONLY payload_statuses USING btree (service_id, status_id);

CREATE INDEX payload_statuses_status_id_created_at_idx ON ONLY payload_statuses USING btree (status_id, created_at);

CREATE INDEX payload_statuses_status_id_date_idx ON ONLY payload_statuses USING btree (status_id, date);

CREATE INDEX payload_statuses_source_id_date_idx ON ONLY payload_statuses USING btree (status_id, date);

CREATE INDEX payloads_account_idx ON payloads USING btree (account);

CREATE INDEX payloads_created_at_idx ON payloads USING btree (created_at);

CREATE UNIQUE INDEX payloads_id_idx ON payloads USING btree (id);

CREATE INDEX payloads_inventory_id_idx ON payloads USING btree (inventory_id);

CREATE UNIQUE INDEX payloads_request_id_idx ON payloads USING btree (request_id);

CREATE INDEX payloads_system_id_idx ON payloads USING btree (system_id);

CREATE UNIQUE INDEX services_id_idx ON services USING btree (id);

CREATE UNIQUE INDEX services_name_idx ON services USING btree (name);

CREATE UNIQUE INDEX sources_id_idx ON sources USING btree (id);

CREATE UNIQUE INDEX sources_name_idx ON sources USING btree (name);

CREATE UNIQUE INDEX statuses_id_idx ON statuses USING btree (id);

CREATE UNIQUE INDEX statuses_name_idx ON statuses USING btree (name);


ALTER TABLE payload_statuses
    ADD CONSTRAINT fk_payload_statuses_payload FOREIGN KEY (payload_id) REFERENCES payloads(id);

ALTER TABLE payload_statuses
    ADD CONSTRAINT fk_payload_statuses_service FOREIGN KEY (service_id) REFERENCES services(id);

ALTER TABLE payload_statuses
    ADD CONSTRAINT fk_payload_statuses_source FOREIGN KEY (source_id) REFERENCES sources(id);

ALTER TABLE payload_statuses
    ADD CONSTRAINT fk_payload_statuses_status FOREIGN KEY (status_id) REFERENCES statuses(id);

ALTER TABLE payload_statuses
    ADD CONSTRAINT payload_statuses_payload_id_fkey FOREIGN KEY (payload_id) REFERENCES payloads(id) ON DELETE CASCADE;

ALTER TABLE payload_statuses
    ADD CONSTRAINT payload_statuses_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id);

ALTER TABLE payload_statuses
    ADD CONSTRAINT payload_statuses_source_id_fkey FOREIGN KEY (source_id) REFERENCES sources(id);

ALTER TABLE payload_statuses
    ADD CONSTRAINT payload_statuses_status_id_fkey FOREIGN KEY (status_id) REFERENCES statuses(id);


SELECT create_partition(NOW()::DATE, NOW()::DATE + interval '1 DAY');
