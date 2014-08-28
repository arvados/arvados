--
-- PostgreSQL database dump
--

SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET search_path = public, pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: api_client_authorizations; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE api_client_authorizations (
    id integer NOT NULL,
    api_token character varying(255) NOT NULL,
    api_client_id integer NOT NULL,
    user_id integer NOT NULL,
    created_by_ip_address character varying(255),
    last_used_by_ip_address character varying(255),
    last_used_at timestamp without time zone,
    expires_at timestamp without time zone,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    default_owner_uuid character varying(255),
    scopes text DEFAULT '---
- all
'::text NOT NULL
);


--
-- Name: api_client_authorizations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE api_client_authorizations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: api_client_authorizations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE api_client_authorizations_id_seq OWNED BY api_client_authorizations.id;


--
-- Name: api_clients; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE api_clients (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    name character varying(255),
    url_prefix character varying(255),
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    is_trusted boolean DEFAULT false
);


--
-- Name: api_clients_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE api_clients_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: api_clients_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE api_clients_id_seq OWNED BY api_clients.id;


--
-- Name: authorized_keys; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE authorized_keys (
    id integer NOT NULL,
    uuid character varying(255) NOT NULL,
    owner_uuid character varying(255) NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    name character varying(255),
    key_type character varying(255),
    authorized_user_uuid character varying(255),
    public_key text,
    expires_at timestamp without time zone,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: authorized_keys_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE authorized_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: authorized_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE authorized_keys_id_seq OWNED BY authorized_keys.id;


--
-- Name: collections; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE collections (
    id integer NOT NULL,
    owner_uuid character varying(255),
    created_at timestamp without time zone NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    portable_data_hash character varying(255),
    redundancy integer,
    redundancy_confirmed_by_client_uuid character varying(255),
    redundancy_confirmed_at timestamp without time zone,
    redundancy_confirmed_as integer,
    updated_at timestamp without time zone NOT NULL,
    uuid character varying(255),
    manifest_text text,
    name character varying(255),
    description character varying(255),
    properties text,
    expires_at date
);


--
-- Name: collections_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE collections_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: collections_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE collections_id_seq OWNED BY collections.id;


--
-- Name: commit_ancestors; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE commit_ancestors (
    id integer NOT NULL,
    repository_name character varying(255),
    descendant character varying(255) NOT NULL,
    ancestor character varying(255) NOT NULL,
    "is" boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: commit_ancestors_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE commit_ancestors_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: commit_ancestors_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE commit_ancestors_id_seq OWNED BY commit_ancestors.id;


--
-- Name: commits; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE commits (
    id integer NOT NULL,
    repository_name character varying(255),
    sha1 character varying(255),
    message character varying(255),
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: commits_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE commits_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: commits_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE commits_id_seq OWNED BY commits.id;


--
-- Name: groups; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE groups (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    created_at timestamp without time zone NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    name character varying(255) NOT NULL,
    description text,
    updated_at timestamp without time zone NOT NULL,
    group_class character varying(255)
);


--
-- Name: groups_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: groups_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE groups_id_seq OWNED BY groups.id;


--
-- Name: humans; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE humans (
    id integer NOT NULL,
    uuid character varying(255) NOT NULL,
    owner_uuid character varying(255) NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    properties text,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: humans_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE humans_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: humans_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE humans_id_seq OWNED BY humans.id;


--
-- Name: job_tasks; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE job_tasks (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    job_uuid character varying(255),
    sequence integer,
    parameters text,
    output text,
    progress double precision,
    success boolean,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    created_by_job_task_uuid character varying(255),
    qsequence bigint
);


--
-- Name: job_tasks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE job_tasks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: job_tasks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE job_tasks_id_seq OWNED BY job_tasks.id;


--
-- Name: job_tasks_qsequence_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE job_tasks_qsequence_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: job_tasks_qsequence_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE job_tasks_qsequence_seq OWNED BY job_tasks.qsequence;


--
-- Name: jobs; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE jobs (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    submit_id character varying(255),
    script character varying(255),
    script_version character varying(255),
    script_parameters text,
    cancelled_by_client_uuid character varying(255),
    cancelled_by_user_uuid character varying(255),
    cancelled_at timestamp without time zone,
    started_at timestamp without time zone,
    finished_at timestamp without time zone,
    running boolean,
    success boolean,
    output character varying(255),
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    priority character varying(255),
    is_locked_by_uuid character varying(255),
    log character varying(255),
    tasks_summary text,
    runtime_constraints text,
    nondeterministic boolean,
    repository character varying(255),
    supplied_script_version character varying(255),
    docker_image_locator character varying(255),
    name character varying(255),
    description text
);


--
-- Name: jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE jobs_id_seq OWNED BY jobs.id;


--
-- Name: keep_disks; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE keep_disks (
    id integer NOT NULL,
    uuid character varying(255) NOT NULL,
    owner_uuid character varying(255) NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    ping_secret character varying(255) NOT NULL,
    node_uuid character varying(255),
    filesystem_uuid character varying(255),
    bytes_total integer,
    bytes_free integer,
    is_readable boolean DEFAULT true NOT NULL,
    is_writable boolean DEFAULT true NOT NULL,
    last_read_at timestamp without time zone,
    last_write_at timestamp without time zone,
    last_ping_at timestamp without time zone,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    keep_service_uuid character varying(255)
);


--
-- Name: keep_disks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE keep_disks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: keep_disks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE keep_disks_id_seq OWNED BY keep_disks.id;


--
-- Name: keep_services; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE keep_services (
    id integer NOT NULL,
    uuid character varying(255) NOT NULL,
    owner_uuid character varying(255) NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    service_host character varying(255),
    service_port integer,
    service_ssl_flag boolean,
    service_type character varying(255),
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: keep_services_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE keep_services_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: keep_services_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE keep_services_id_seq OWNED BY keep_services.id;


--
-- Name: links; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE links (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    created_at timestamp without time zone NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    tail_uuid character varying(255),
    link_class character varying(255),
    name character varying(255),
    head_uuid character varying(255),
    properties text,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: links_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE links_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: links_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE links_id_seq OWNED BY links.id;


--
-- Name: logs; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE logs (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    object_uuid character varying(255),
    event_at timestamp without time zone,
    event_type character varying(255),
    summary text,
    properties text,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    modified_at timestamp without time zone,
    object_owner_uuid character varying(255)
);


--
-- Name: logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE logs_id_seq OWNED BY logs.id;


--
-- Name: nodes; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE nodes (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    created_at timestamp without time zone NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    slot_number integer,
    hostname character varying(255),
    domain character varying(255),
    ip_address character varying(255),
    first_ping_at timestamp without time zone,
    last_ping_at timestamp without time zone,
    info text,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: nodes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE nodes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: nodes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE nodes_id_seq OWNED BY nodes.id;


--
-- Name: pipeline_instances; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE pipeline_instances (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    created_at timestamp without time zone NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    pipeline_template_uuid character varying(255),
    name character varying(255),
    components text,
    updated_at timestamp without time zone NOT NULL,
    properties text,
    state character varying(255),
    components_summary text,
    description text
);


--
-- Name: pipeline_instances_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE pipeline_instances_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: pipeline_instances_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE pipeline_instances_id_seq OWNED BY pipeline_instances.id;


--
-- Name: pipeline_templates; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE pipeline_templates (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    created_at timestamp without time zone NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    name character varying(255),
    components text,
    updated_at timestamp without time zone NOT NULL,
    description text
);


--
-- Name: pipeline_templates_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE pipeline_templates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: pipeline_templates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE pipeline_templates_id_seq OWNED BY pipeline_templates.id;


--
-- Name: repositories; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE repositories (
    id integer NOT NULL,
    uuid character varying(255) NOT NULL,
    owner_uuid character varying(255) NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    name character varying(255),
    fetch_url character varying(255),
    push_url character varying(255),
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: repositories_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE repositories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: repositories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE repositories_id_seq OWNED BY repositories.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE schema_migrations (
    version character varying(255) NOT NULL
);


--
-- Name: specimens; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE specimens (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    created_at timestamp without time zone NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    material character varying(255),
    updated_at timestamp without time zone NOT NULL,
    properties text
);


--
-- Name: specimens_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE specimens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: specimens_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE specimens_id_seq OWNED BY specimens.id;


--
-- Name: traits; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE traits (
    id integer NOT NULL,
    uuid character varying(255) NOT NULL,
    owner_uuid character varying(255) NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    name character varying(255),
    properties text,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: traits_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE traits_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: traits_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE traits_id_seq OWNED BY traits.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE users (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    created_at timestamp without time zone NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    email character varying(255),
    first_name character varying(255),
    last_name character varying(255),
    identity_url character varying(255),
    is_admin boolean,
    prefs text,
    updated_at timestamp without time zone NOT NULL,
    default_owner_uuid character varying(255),
    is_active boolean DEFAULT false
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE users_id_seq OWNED BY users.id;


--
-- Name: virtual_machines; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE virtual_machines (
    id integer NOT NULL,
    uuid character varying(255) NOT NULL,
    owner_uuid character varying(255) NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    hostname character varying(255),
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: virtual_machines_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE virtual_machines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: virtual_machines_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE virtual_machines_id_seq OWNED BY virtual_machines.id;


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY api_client_authorizations ALTER COLUMN id SET DEFAULT nextval('api_client_authorizations_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY api_clients ALTER COLUMN id SET DEFAULT nextval('api_clients_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY authorized_keys ALTER COLUMN id SET DEFAULT nextval('authorized_keys_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY collections ALTER COLUMN id SET DEFAULT nextval('collections_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY commit_ancestors ALTER COLUMN id SET DEFAULT nextval('commit_ancestors_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY commits ALTER COLUMN id SET DEFAULT nextval('commits_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY groups ALTER COLUMN id SET DEFAULT nextval('groups_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY humans ALTER COLUMN id SET DEFAULT nextval('humans_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY job_tasks ALTER COLUMN id SET DEFAULT nextval('job_tasks_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY jobs ALTER COLUMN id SET DEFAULT nextval('jobs_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY keep_disks ALTER COLUMN id SET DEFAULT nextval('keep_disks_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY keep_services ALTER COLUMN id SET DEFAULT nextval('keep_services_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY links ALTER COLUMN id SET DEFAULT nextval('links_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY logs ALTER COLUMN id SET DEFAULT nextval('logs_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY nodes ALTER COLUMN id SET DEFAULT nextval('nodes_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY pipeline_instances ALTER COLUMN id SET DEFAULT nextval('pipeline_instances_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY pipeline_templates ALTER COLUMN id SET DEFAULT nextval('pipeline_templates_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY repositories ALTER COLUMN id SET DEFAULT nextval('repositories_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY specimens ALTER COLUMN id SET DEFAULT nextval('specimens_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY traits ALTER COLUMN id SET DEFAULT nextval('traits_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY virtual_machines ALTER COLUMN id SET DEFAULT nextval('virtual_machines_id_seq'::regclass);


--
-- Name: api_client_authorizations_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY api_client_authorizations
    ADD CONSTRAINT api_client_authorizations_pkey PRIMARY KEY (id);


--
-- Name: api_clients_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY api_clients
    ADD CONSTRAINT api_clients_pkey PRIMARY KEY (id);


--
-- Name: authorized_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY authorized_keys
    ADD CONSTRAINT authorized_keys_pkey PRIMARY KEY (id);


--
-- Name: collections_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY collections
    ADD CONSTRAINT collections_pkey PRIMARY KEY (id);


--
-- Name: commit_ancestors_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY commit_ancestors
    ADD CONSTRAINT commit_ancestors_pkey PRIMARY KEY (id);


--
-- Name: commits_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY commits
    ADD CONSTRAINT commits_pkey PRIMARY KEY (id);


--
-- Name: groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (id);


--
-- Name: humans_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY humans
    ADD CONSTRAINT humans_pkey PRIMARY KEY (id);


--
-- Name: job_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY job_tasks
    ADD CONSTRAINT job_tasks_pkey PRIMARY KEY (id);


--
-- Name: jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY jobs
    ADD CONSTRAINT jobs_pkey PRIMARY KEY (id);


--
-- Name: keep_disks_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY keep_disks
    ADD CONSTRAINT keep_disks_pkey PRIMARY KEY (id);


--
-- Name: keep_services_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY keep_services
    ADD CONSTRAINT keep_services_pkey PRIMARY KEY (id);


--
-- Name: links_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY links
    ADD CONSTRAINT links_pkey PRIMARY KEY (id);


--
-- Name: logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY logs
    ADD CONSTRAINT logs_pkey PRIMARY KEY (id);


--
-- Name: nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY nodes
    ADD CONSTRAINT nodes_pkey PRIMARY KEY (id);


--
-- Name: pipeline_instances_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY pipeline_instances
    ADD CONSTRAINT pipeline_instances_pkey PRIMARY KEY (id);


--
-- Name: pipeline_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY pipeline_templates
    ADD CONSTRAINT pipeline_templates_pkey PRIMARY KEY (id);


--
-- Name: repositories_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY repositories
    ADD CONSTRAINT repositories_pkey PRIMARY KEY (id);


--
-- Name: specimens_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY specimens
    ADD CONSTRAINT specimens_pkey PRIMARY KEY (id);


--
-- Name: traits_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY traits
    ADD CONSTRAINT traits_pkey PRIMARY KEY (id);


--
-- Name: users_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: virtual_machines_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY virtual_machines
    ADD CONSTRAINT virtual_machines_pkey PRIMARY KEY (id);


--
-- Name: collection_owner_uuid_name_unique; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX collection_owner_uuid_name_unique ON collections USING btree (owner_uuid, name);


--
-- Name: groups_owner_uuid_name_unique; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX groups_owner_uuid_name_unique ON groups USING btree (owner_uuid, name);


--
-- Name: index_api_client_authorizations_on_api_client_id; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_api_client_authorizations_on_api_client_id ON api_client_authorizations USING btree (api_client_id);


--
-- Name: index_api_client_authorizations_on_api_token; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_api_client_authorizations_on_api_token ON api_client_authorizations USING btree (api_token);


--
-- Name: index_api_client_authorizations_on_expires_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_api_client_authorizations_on_expires_at ON api_client_authorizations USING btree (expires_at);


--
-- Name: index_api_client_authorizations_on_user_id; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_api_client_authorizations_on_user_id ON api_client_authorizations USING btree (user_id);


--
-- Name: index_api_clients_on_created_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_api_clients_on_created_at ON api_clients USING btree (created_at);


--
-- Name: index_api_clients_on_modified_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_api_clients_on_modified_at ON api_clients USING btree (modified_at);


--
-- Name: index_api_clients_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_api_clients_on_uuid ON api_clients USING btree (uuid);


--
-- Name: index_authkeys_on_user_and_expires_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_authkeys_on_user_and_expires_at ON authorized_keys USING btree (authorized_user_uuid, expires_at);


--
-- Name: index_authorized_keys_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_authorized_keys_on_uuid ON authorized_keys USING btree (uuid);


--
-- Name: index_collections_on_created_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_collections_on_created_at ON collections USING btree (created_at);


--
-- Name: index_collections_on_modified_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_collections_on_modified_at ON collections USING btree (modified_at);


--
-- Name: index_collections_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_collections_on_uuid ON collections USING btree (uuid);


--
-- Name: index_commit_ancestors_on_descendant_and_ancestor; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_commit_ancestors_on_descendant_and_ancestor ON commit_ancestors USING btree (descendant, ancestor);


--
-- Name: index_commits_on_repository_name_and_sha1; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_commits_on_repository_name_and_sha1 ON commits USING btree (repository_name, sha1);


--
-- Name: index_groups_on_created_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_groups_on_created_at ON groups USING btree (created_at);


--
-- Name: index_groups_on_group_class; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_groups_on_group_class ON groups USING btree (group_class);


--
-- Name: index_groups_on_modified_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_groups_on_modified_at ON groups USING btree (modified_at);


--
-- Name: index_groups_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_groups_on_uuid ON groups USING btree (uuid);


--
-- Name: index_humans_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_humans_on_uuid ON humans USING btree (uuid);


--
-- Name: index_job_tasks_on_created_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_job_tasks_on_created_at ON job_tasks USING btree (created_at);


--
-- Name: index_job_tasks_on_job_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_job_tasks_on_job_uuid ON job_tasks USING btree (job_uuid);


--
-- Name: index_job_tasks_on_modified_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_job_tasks_on_modified_at ON job_tasks USING btree (modified_at);


--
-- Name: index_job_tasks_on_sequence; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_job_tasks_on_sequence ON job_tasks USING btree (sequence);


--
-- Name: index_job_tasks_on_success; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_job_tasks_on_success ON job_tasks USING btree (success);


--
-- Name: index_job_tasks_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_job_tasks_on_uuid ON job_tasks USING btree (uuid);


--
-- Name: index_jobs_on_created_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_jobs_on_created_at ON jobs USING btree (created_at);


--
-- Name: index_jobs_on_finished_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_jobs_on_finished_at ON jobs USING btree (finished_at);


--
-- Name: index_jobs_on_modified_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_jobs_on_modified_at ON jobs USING btree (modified_at);


--
-- Name: index_jobs_on_output; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_jobs_on_output ON jobs USING btree (output);


--
-- Name: index_jobs_on_script; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_jobs_on_script ON jobs USING btree (script);


--
-- Name: index_jobs_on_started_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_jobs_on_started_at ON jobs USING btree (started_at);


--
-- Name: index_jobs_on_submit_id; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_jobs_on_submit_id ON jobs USING btree (submit_id);


--
-- Name: index_jobs_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_jobs_on_uuid ON jobs USING btree (uuid);


--
-- Name: index_keep_disks_on_filesystem_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_keep_disks_on_filesystem_uuid ON keep_disks USING btree (filesystem_uuid);


--
-- Name: index_keep_disks_on_last_ping_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_keep_disks_on_last_ping_at ON keep_disks USING btree (last_ping_at);


--
-- Name: index_keep_disks_on_node_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_keep_disks_on_node_uuid ON keep_disks USING btree (node_uuid);


--
-- Name: index_keep_disks_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_keep_disks_on_uuid ON keep_disks USING btree (uuid);


--
-- Name: index_keep_services_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_keep_services_on_uuid ON keep_services USING btree (uuid);


--
-- Name: index_links_on_created_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_links_on_created_at ON links USING btree (created_at);


--
-- Name: index_links_on_head_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_links_on_head_uuid ON links USING btree (head_uuid);


--
-- Name: index_links_on_modified_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_links_on_modified_at ON links USING btree (modified_at);


--
-- Name: index_links_on_tail_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_links_on_tail_uuid ON links USING btree (tail_uuid);


--
-- Name: index_links_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_links_on_uuid ON links USING btree (uuid);


--
-- Name: index_logs_on_created_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_logs_on_created_at ON logs USING btree (created_at);


--
-- Name: index_logs_on_event_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_logs_on_event_at ON logs USING btree (event_at);


--
-- Name: index_logs_on_event_type; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_logs_on_event_type ON logs USING btree (event_type);


--
-- Name: index_logs_on_modified_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_logs_on_modified_at ON logs USING btree (modified_at);


--
-- Name: index_logs_on_object_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_logs_on_object_uuid ON logs USING btree (object_uuid);


--
-- Name: index_logs_on_summary; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_logs_on_summary ON logs USING btree (summary);


--
-- Name: index_logs_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_logs_on_uuid ON logs USING btree (uuid);


--
-- Name: index_nodes_on_created_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_nodes_on_created_at ON nodes USING btree (created_at);


--
-- Name: index_nodes_on_hostname; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_nodes_on_hostname ON nodes USING btree (hostname);


--
-- Name: index_nodes_on_modified_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_nodes_on_modified_at ON nodes USING btree (modified_at);


--
-- Name: index_nodes_on_slot_number; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_nodes_on_slot_number ON nodes USING btree (slot_number);


--
-- Name: index_nodes_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_nodes_on_uuid ON nodes USING btree (uuid);


--
-- Name: index_pipeline_instances_on_created_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_pipeline_instances_on_created_at ON pipeline_instances USING btree (created_at);


--
-- Name: index_pipeline_instances_on_modified_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_pipeline_instances_on_modified_at ON pipeline_instances USING btree (modified_at);


--
-- Name: index_pipeline_instances_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_pipeline_instances_on_uuid ON pipeline_instances USING btree (uuid);


--
-- Name: index_pipeline_templates_on_created_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_pipeline_templates_on_created_at ON pipeline_templates USING btree (created_at);


--
-- Name: index_pipeline_templates_on_modified_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_pipeline_templates_on_modified_at ON pipeline_templates USING btree (modified_at);


--
-- Name: index_pipeline_templates_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_pipeline_templates_on_uuid ON pipeline_templates USING btree (uuid);


--
-- Name: index_repositories_on_name; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_repositories_on_name ON repositories USING btree (name);


--
-- Name: index_repositories_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_repositories_on_uuid ON repositories USING btree (uuid);


--
-- Name: index_specimens_on_created_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_specimens_on_created_at ON specimens USING btree (created_at);


--
-- Name: index_specimens_on_modified_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_specimens_on_modified_at ON specimens USING btree (modified_at);


--
-- Name: index_specimens_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_specimens_on_uuid ON specimens USING btree (uuid);


--
-- Name: index_traits_on_name; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_traits_on_name ON traits USING btree (name);


--
-- Name: index_traits_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_traits_on_uuid ON traits USING btree (uuid);


--
-- Name: index_users_on_created_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_users_on_created_at ON users USING btree (created_at);


--
-- Name: index_users_on_modified_at; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_users_on_modified_at ON users USING btree (modified_at);


--
-- Name: index_users_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_users_on_uuid ON users USING btree (uuid);


--
-- Name: index_virtual_machines_on_hostname; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX index_virtual_machines_on_hostname ON virtual_machines USING btree (hostname);


--
-- Name: index_virtual_machines_on_uuid; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX index_virtual_machines_on_uuid ON virtual_machines USING btree (uuid);


--
-- Name: jobs_owner_uuid_name_unique; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX jobs_owner_uuid_name_unique ON jobs USING btree (owner_uuid, name);


--
-- Name: links_tail_name_unique_if_link_class_name; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX links_tail_name_unique_if_link_class_name ON links USING btree (tail_uuid, name) WHERE ((link_class)::text = 'name'::text);


--
-- Name: pipeline_instance_owner_uuid_name_unique; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX pipeline_instance_owner_uuid_name_unique ON pipeline_instances USING btree (owner_uuid, name);


--
-- Name: pipeline_template_owner_uuid_name_unique; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX pipeline_template_owner_uuid_name_unique ON pipeline_templates USING btree (owner_uuid, name);


--
-- Name: unique_schema_migrations; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX unique_schema_migrations ON schema_migrations USING btree (version);


--
-- PostgreSQL database dump complete
--

SET search_path TO "$user",public;

INSERT INTO schema_migrations (version) VALUES ('20121016005009');

INSERT INTO schema_migrations (version) VALUES ('20130105203021');

INSERT INTO schema_migrations (version) VALUES ('20130105224358');

INSERT INTO schema_migrations (version) VALUES ('20130105224618');

INSERT INTO schema_migrations (version) VALUES ('20130107181109');

INSERT INTO schema_migrations (version) VALUES ('20130107212832');

INSERT INTO schema_migrations (version) VALUES ('20130109175700');

INSERT INTO schema_migrations (version) VALUES ('20130109220548');

INSERT INTO schema_migrations (version) VALUES ('20130113214204');

INSERT INTO schema_migrations (version) VALUES ('20130116024233');

INSERT INTO schema_migrations (version) VALUES ('20130116215213');

INSERT INTO schema_migrations (version) VALUES ('20130118002239');

INSERT INTO schema_migrations (version) VALUES ('20130122020042');

INSERT INTO schema_migrations (version) VALUES ('20130122201442');

INSERT INTO schema_migrations (version) VALUES ('20130122221616');

INSERT INTO schema_migrations (version) VALUES ('20130123174514');

INSERT INTO schema_migrations (version) VALUES ('20130123180224');

INSERT INTO schema_migrations (version) VALUES ('20130123180228');

INSERT INTO schema_migrations (version) VALUES ('20130125220425');

INSERT INTO schema_migrations (version) VALUES ('20130128202518');

INSERT INTO schema_migrations (version) VALUES ('20130128231343');

INSERT INTO schema_migrations (version) VALUES ('20130130205749');

INSERT INTO schema_migrations (version) VALUES ('20130203104818');

INSERT INTO schema_migrations (version) VALUES ('20130203104824');

INSERT INTO schema_migrations (version) VALUES ('20130203115329');

INSERT INTO schema_migrations (version) VALUES ('20130207195855');

INSERT INTO schema_migrations (version) VALUES ('20130218181504');

INSERT INTO schema_migrations (version) VALUES ('20130226170000');

INSERT INTO schema_migrations (version) VALUES ('20130313175417');

INSERT INTO schema_migrations (version) VALUES ('20130315155820');

INSERT INTO schema_migrations (version) VALUES ('20130315183626');

INSERT INTO schema_migrations (version) VALUES ('20130315213205');

INSERT INTO schema_migrations (version) VALUES ('20130318002138');

INSERT INTO schema_migrations (version) VALUES ('20130319165853');

INSERT INTO schema_migrations (version) VALUES ('20130319180730');

INSERT INTO schema_migrations (version) VALUES ('20130319194637');

INSERT INTO schema_migrations (version) VALUES ('20130319201431');

INSERT INTO schema_migrations (version) VALUES ('20130319235957');

INSERT INTO schema_migrations (version) VALUES ('20130320000107');

INSERT INTO schema_migrations (version) VALUES ('20130326173804');

INSERT INTO schema_migrations (version) VALUES ('20130326182917');

INSERT INTO schema_migrations (version) VALUES ('20130415020241');

INSERT INTO schema_migrations (version) VALUES ('20130425024459');

INSERT INTO schema_migrations (version) VALUES ('20130425214427');

INSERT INTO schema_migrations (version) VALUES ('20130523060112');

INSERT INTO schema_migrations (version) VALUES ('20130523060213');

INSERT INTO schema_migrations (version) VALUES ('20130524042319');

INSERT INTO schema_migrations (version) VALUES ('20130528134100');

INSERT INTO schema_migrations (version) VALUES ('20130606183519');

INSERT INTO schema_migrations (version) VALUES ('20130608053730');

INSERT INTO schema_migrations (version) VALUES ('20130610202538');

INSERT INTO schema_migrations (version) VALUES ('20130611163736');

INSERT INTO schema_migrations (version) VALUES ('20130612042554');

INSERT INTO schema_migrations (version) VALUES ('20130617150007');

INSERT INTO schema_migrations (version) VALUES ('20130626002829');

INSERT INTO schema_migrations (version) VALUES ('20130626022810');

INSERT INTO schema_migrations (version) VALUES ('20130627154537');

INSERT INTO schema_migrations (version) VALUES ('20130627184333');

INSERT INTO schema_migrations (version) VALUES ('20130708163414');

INSERT INTO schema_migrations (version) VALUES ('20130708182912');

INSERT INTO schema_migrations (version) VALUES ('20130708185153');

INSERT INTO schema_migrations (version) VALUES ('20130724153034');

INSERT INTO schema_migrations (version) VALUES ('20131007180607');

INSERT INTO schema_migrations (version) VALUES ('20140117231056');

INSERT INTO schema_migrations (version) VALUES ('20140124222114');

INSERT INTO schema_migrations (version) VALUES ('20140129184311');

INSERT INTO schema_migrations (version) VALUES ('20140317135600');

INSERT INTO schema_migrations (version) VALUES ('20140319160547');

INSERT INTO schema_migrations (version) VALUES ('20140321191343');

INSERT INTO schema_migrations (version) VALUES ('20140324024606');

INSERT INTO schema_migrations (version) VALUES ('20140325175653');

INSERT INTO schema_migrations (version) VALUES ('20140402001908');

INSERT INTO schema_migrations (version) VALUES ('20140407184311');

INSERT INTO schema_migrations (version) VALUES ('20140421140924');

INSERT INTO schema_migrations (version) VALUES ('20140421151939');

INSERT INTO schema_migrations (version) VALUES ('20140421151940');

INSERT INTO schema_migrations (version) VALUES ('20140422011506');

INSERT INTO schema_migrations (version) VALUES ('20140423132913');

INSERT INTO schema_migrations (version) VALUES ('20140423133559');

INSERT INTO schema_migrations (version) VALUES ('20140501165548');

INSERT INTO schema_migrations (version) VALUES ('20140519205916');

INSERT INTO schema_migrations (version) VALUES ('20140527152921');

INSERT INTO schema_migrations (version) VALUES ('20140530200539');

INSERT INTO schema_migrations (version) VALUES ('20140601022548');

INSERT INTO schema_migrations (version) VALUES ('20140602143352');

INSERT INTO schema_migrations (version) VALUES ('20140607150616');

INSERT INTO schema_migrations (version) VALUES ('20140611173003');

INSERT INTO schema_migrations (version) VALUES ('20140627210837');

INSERT INTO schema_migrations (version) VALUES ('20140709172343');

INSERT INTO schema_migrations (version) VALUES ('20140714184006');

INSERT INTO schema_migrations (version) VALUES ('20140811184643');

INSERT INTO schema_migrations (version) VALUES ('20140815171049');

INSERT INTO schema_migrations (version) VALUES ('20140817035914');

INSERT INTO schema_migrations (version) VALUES ('20140818125735');

INSERT INTO schema_migrations (version) VALUES ('20140826180337');