-- Copyright (C) The Arvados Authors. All rights reserved.
--
-- SPDX-License-Identifier: AGPL-3.0

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

-- COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET search_path = public, pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: api_client_authorizations; Type: TABLE; Schema: public; Owner: -
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
    scopes text DEFAULT '["all"]'::text,
    uuid character varying(255) NOT NULL
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
-- Name: api_clients; Type: TABLE; Schema: public; Owner: -
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
-- Name: authorized_keys; Type: TABLE; Schema: public; Owner: -
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
-- Name: collections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE collections (
    id integer NOT NULL,
    owner_uuid character varying(255),
    created_at timestamp without time zone NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    portable_data_hash character varying(255),
    replication_desired integer,
    replication_confirmed_at timestamp without time zone,
    replication_confirmed integer,
    updated_at timestamp without time zone NOT NULL,
    uuid character varying(255),
    manifest_text text,
    name character varying(255),
    description character varying(524288),
    properties jsonb,
    delete_at timestamp without time zone,
    file_names character varying(8192),
    trash_at timestamp without time zone,
    is_trashed boolean DEFAULT false NOT NULL,
    storage_classes_desired jsonb DEFAULT '["default"]'::jsonb,
    storage_classes_confirmed jsonb DEFAULT '[]'::jsonb,
    storage_classes_confirmed_at timestamp without time zone
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
-- Name: commit_ancestors; Type: TABLE; Schema: public; Owner: -
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
-- Name: commits; Type: TABLE; Schema: public; Owner: -
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
-- Name: container_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE container_requests (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    created_at timestamp without time zone NOT NULL,
    modified_at timestamp without time zone,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    name character varying(255),
    description text,
    properties jsonb,
    state character varying(255),
    requesting_container_uuid character varying(255),
    container_uuid character varying(255),
    container_count_max integer,
    mounts text,
    runtime_constraints text,
    container_image character varying(255),
    environment text,
    cwd character varying(255),
    command text,
    output_path character varying(255),
    priority integer,
    expires_at timestamp without time zone,
    filters text,
    updated_at timestamp without time zone NOT NULL,
    container_count integer DEFAULT 0,
    use_existing boolean DEFAULT true,
    scheduling_parameters text,
    output_uuid character varying(255),
    log_uuid character varying(255),
    output_name character varying(255) DEFAULT NULL::character varying,
    output_ttl integer DEFAULT 0 NOT NULL,
    secret_mounts jsonb DEFAULT '{}'::jsonb
);


--
-- Name: container_requests_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE container_requests_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: container_requests_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE container_requests_id_seq OWNED BY container_requests.id;


--
-- Name: containers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE containers (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    created_at timestamp without time zone NOT NULL,
    modified_at timestamp without time zone,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    state character varying(255),
    started_at timestamp without time zone,
    finished_at timestamp without time zone,
    log character varying(255),
    environment text,
    cwd character varying(255),
    command text,
    output_path character varying(255),
    mounts text,
    runtime_constraints text,
    output character varying(255),
    container_image character varying(255),
    progress double precision,
    priority bigint,
    updated_at timestamp without time zone NOT NULL,
    exit_code integer,
    auth_uuid character varying(255),
    locked_by_uuid character varying(255),
    scheduling_parameters text,
    secret_mounts jsonb DEFAULT '{}'::jsonb,
    secret_mounts_md5 character varying DEFAULT '99914b932bd37a50b983c5e7c90ae93b'::character varying
);


--
-- Name: containers_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE containers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: containers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE containers_id_seq OWNED BY containers.id;


--
-- Name: groups; Type: TABLE; Schema: public; Owner: -
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
    description character varying(524288),
    updated_at timestamp without time zone NOT NULL,
    group_class character varying(255),
    trash_at timestamp without time zone,
    is_trashed boolean DEFAULT false NOT NULL,
    delete_at timestamp without time zone,
    properties jsonb DEFAULT '{}'::jsonb
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
-- Name: humans; Type: TABLE; Schema: public; Owner: -
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
-- Name: job_tasks; Type: TABLE; Schema: public; Owner: -
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
    qsequence bigint,
    started_at timestamp without time zone,
    finished_at timestamp without time zone
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
-- Name: jobs; Type: TABLE; Schema: public; Owner: -
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
    is_locked_by_uuid character varying(255),
    log character varying(255),
    tasks_summary text,
    runtime_constraints text,
    nondeterministic boolean,
    repository character varying(255),
    supplied_script_version character varying(255),
    docker_image_locator character varying(255),
    priority integer DEFAULT 0 NOT NULL,
    description character varying(524288),
    state character varying(255),
    arvados_sdk_version character varying(255),
    components text,
    script_parameters_digest character varying(255)
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
-- Name: keep_disks; Type: TABLE; Schema: public; Owner: -
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
-- Name: keep_services; Type: TABLE; Schema: public; Owner: -
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
    updated_at timestamp without time zone NOT NULL,
    read_only boolean DEFAULT false NOT NULL
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
-- Name: links; Type: TABLE; Schema: public; Owner: -
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
    properties jsonb,
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
-- Name: logs; Type: TABLE; Schema: public; Owner: -
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
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE users (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255) NOT NULL,
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
    is_active boolean DEFAULT false,
    username character varying(255),
    redirect_to_user_uuid character varying
);


--
-- Name: materialized_permission_view; Type: MATERIALIZED VIEW; Schema: public; Owner: -
--

CREATE MATERIALIZED VIEW materialized_permission_view AS
 WITH RECURSIVE perm_value(name, val) AS (
         VALUES ('can_read'::text,(1)::smallint), ('can_login'::text,1), ('can_write'::text,2), ('can_manage'::text,3)
        ), perm_edges(tail_uuid, head_uuid, val, follow, trashed) AS (
         SELECT links.tail_uuid,
            links.head_uuid,
            pv.val,
            ((pv.val = 3) OR (groups.uuid IS NOT NULL)) AS follow,
            (0)::smallint AS trashed,
            (0)::smallint AS followtrash
           FROM ((links
             LEFT JOIN perm_value pv ON ((pv.name = (links.name)::text)))
             LEFT JOIN groups ON (((pv.val < 3) AND ((groups.uuid)::text = (links.head_uuid)::text))))
          WHERE ((links.link_class)::text = 'permission'::text)
        UNION ALL
         SELECT groups.owner_uuid,
            groups.uuid,
            3,
            true AS bool,
                CASE
                    WHEN ((groups.trash_at IS NOT NULL) AND (groups.trash_at < clock_timestamp())) THEN 1
                    ELSE 0
                END AS "case",
            1
           FROM groups
        ), perm(val, follow, user_uuid, target_uuid, trashed) AS (
         SELECT (3)::smallint AS val,
            true AS follow,
            (users.uuid)::character varying(32) AS user_uuid,
            (users.uuid)::character varying(32) AS target_uuid,
            (0)::smallint AS trashed
           FROM users
        UNION
         SELECT (LEAST((perm_1.val)::integer, edges.val))::smallint AS val,
            edges.follow,
            perm_1.user_uuid,
            (edges.head_uuid)::character varying(32) AS target_uuid,
            ((GREATEST((perm_1.trashed)::integer, edges.trashed) * edges.followtrash))::smallint AS trashed
           FROM (perm perm_1
             JOIN perm_edges edges ON ((perm_1.follow AND ((edges.tail_uuid)::text = (perm_1.target_uuid)::text))))
        )
 SELECT perm.user_uuid,
    perm.target_uuid,
    max(perm.val) AS perm_level,
        CASE perm.follow
            WHEN true THEN perm.target_uuid
            ELSE NULL::character varying
        END AS target_owner_uuid,
    max(perm.trashed) AS trashed
   FROM perm
  GROUP BY perm.user_uuid, perm.target_uuid,
        CASE perm.follow
            WHEN true THEN perm.target_uuid
            ELSE NULL::character varying
        END
  WITH NO DATA;


--
-- Name: nodes; Type: TABLE; Schema: public; Owner: -
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
    info jsonb,
    updated_at timestamp without time zone NOT NULL,
    properties jsonb,
    job_uuid character varying(255)
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
-- Name: permission_refresh_lock; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE permission_refresh_lock (
    id integer NOT NULL
);


--
-- Name: permission_refresh_lock_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE permission_refresh_lock_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: permission_refresh_lock_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE permission_refresh_lock_id_seq OWNED BY permission_refresh_lock.id;


--
-- Name: pipeline_instances; Type: TABLE; Schema: public; Owner: -
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
    started_at timestamp without time zone,
    finished_at timestamp without time zone,
    description character varying(524288)
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
-- Name: pipeline_templates; Type: TABLE; Schema: public; Owner: -
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
    description character varying(524288)
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
-- Name: repositories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE repositories (
    id integer NOT NULL,
    uuid character varying(255) NOT NULL,
    owner_uuid character varying(255) NOT NULL,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    modified_at timestamp without time zone,
    name character varying(255),
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
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE schema_migrations (
    version character varying(255) NOT NULL
);


--
-- Name: specimens; Type: TABLE; Schema: public; Owner: -
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
-- Name: traits; Type: TABLE; Schema: public; Owner: -
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
-- Name: virtual_machines; Type: TABLE; Schema: public; Owner: -
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
-- Name: workflows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE workflows (
    id integer NOT NULL,
    uuid character varying(255),
    owner_uuid character varying(255),
    created_at timestamp without time zone NOT NULL,
    modified_at timestamp without time zone,
    modified_by_client_uuid character varying(255),
    modified_by_user_uuid character varying(255),
    name character varying(255),
    description text,
    definition text,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: workflows_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE workflows_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: workflows_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE workflows_id_seq OWNED BY workflows.id;


--
-- Name: api_client_authorizations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY api_client_authorizations ALTER COLUMN id SET DEFAULT nextval('api_client_authorizations_id_seq'::regclass);


--
-- Name: api_clients id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY api_clients ALTER COLUMN id SET DEFAULT nextval('api_clients_id_seq'::regclass);


--
-- Name: authorized_keys id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY authorized_keys ALTER COLUMN id SET DEFAULT nextval('authorized_keys_id_seq'::regclass);


--
-- Name: collections id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY collections ALTER COLUMN id SET DEFAULT nextval('collections_id_seq'::regclass);


--
-- Name: commit_ancestors id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY commit_ancestors ALTER COLUMN id SET DEFAULT nextval('commit_ancestors_id_seq'::regclass);


--
-- Name: commits id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY commits ALTER COLUMN id SET DEFAULT nextval('commits_id_seq'::regclass);


--
-- Name: container_requests id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY container_requests ALTER COLUMN id SET DEFAULT nextval('container_requests_id_seq'::regclass);


--
-- Name: containers id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY containers ALTER COLUMN id SET DEFAULT nextval('containers_id_seq'::regclass);


--
-- Name: groups id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY groups ALTER COLUMN id SET DEFAULT nextval('groups_id_seq'::regclass);


--
-- Name: humans id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY humans ALTER COLUMN id SET DEFAULT nextval('humans_id_seq'::regclass);


--
-- Name: job_tasks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY job_tasks ALTER COLUMN id SET DEFAULT nextval('job_tasks_id_seq'::regclass);


--
-- Name: jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY jobs ALTER COLUMN id SET DEFAULT nextval('jobs_id_seq'::regclass);


--
-- Name: keep_disks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY keep_disks ALTER COLUMN id SET DEFAULT nextval('keep_disks_id_seq'::regclass);


--
-- Name: keep_services id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY keep_services ALTER COLUMN id SET DEFAULT nextval('keep_services_id_seq'::regclass);


--
-- Name: links id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY links ALTER COLUMN id SET DEFAULT nextval('links_id_seq'::regclass);


--
-- Name: logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY logs ALTER COLUMN id SET DEFAULT nextval('logs_id_seq'::regclass);


--
-- Name: nodes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY nodes ALTER COLUMN id SET DEFAULT nextval('nodes_id_seq'::regclass);


--
-- Name: permission_refresh_lock id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY permission_refresh_lock ALTER COLUMN id SET DEFAULT nextval('permission_refresh_lock_id_seq'::regclass);


--
-- Name: pipeline_instances id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY pipeline_instances ALTER COLUMN id SET DEFAULT nextval('pipeline_instances_id_seq'::regclass);


--
-- Name: pipeline_templates id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY pipeline_templates ALTER COLUMN id SET DEFAULT nextval('pipeline_templates_id_seq'::regclass);


--
-- Name: repositories id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY repositories ALTER COLUMN id SET DEFAULT nextval('repositories_id_seq'::regclass);


--
-- Name: specimens id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY specimens ALTER COLUMN id SET DEFAULT nextval('specimens_id_seq'::regclass);


--
-- Name: traits id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY traits ALTER COLUMN id SET DEFAULT nextval('traits_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);


--
-- Name: virtual_machines id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY virtual_machines ALTER COLUMN id SET DEFAULT nextval('virtual_machines_id_seq'::regclass);


--
-- Name: workflows id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY workflows ALTER COLUMN id SET DEFAULT nextval('workflows_id_seq'::regclass);


--
-- Name: api_client_authorizations api_client_authorizations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY api_client_authorizations
    ADD CONSTRAINT api_client_authorizations_pkey PRIMARY KEY (id);


--
-- Name: api_clients api_clients_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY api_clients
    ADD CONSTRAINT api_clients_pkey PRIMARY KEY (id);


--
-- Name: authorized_keys authorized_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY authorized_keys
    ADD CONSTRAINT authorized_keys_pkey PRIMARY KEY (id);


--
-- Name: collections collections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY collections
    ADD CONSTRAINT collections_pkey PRIMARY KEY (id);


--
-- Name: commit_ancestors commit_ancestors_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY commit_ancestors
    ADD CONSTRAINT commit_ancestors_pkey PRIMARY KEY (id);


--
-- Name: commits commits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY commits
    ADD CONSTRAINT commits_pkey PRIMARY KEY (id);


--
-- Name: container_requests container_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY container_requests
    ADD CONSTRAINT container_requests_pkey PRIMARY KEY (id);


--
-- Name: containers containers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY containers
    ADD CONSTRAINT containers_pkey PRIMARY KEY (id);


--
-- Name: groups groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (id);


--
-- Name: humans humans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY humans
    ADD CONSTRAINT humans_pkey PRIMARY KEY (id);


--
-- Name: job_tasks job_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY job_tasks
    ADD CONSTRAINT job_tasks_pkey PRIMARY KEY (id);


--
-- Name: jobs jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY jobs
    ADD CONSTRAINT jobs_pkey PRIMARY KEY (id);


--
-- Name: keep_disks keep_disks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY keep_disks
    ADD CONSTRAINT keep_disks_pkey PRIMARY KEY (id);


--
-- Name: keep_services keep_services_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY keep_services
    ADD CONSTRAINT keep_services_pkey PRIMARY KEY (id);


--
-- Name: links links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY links
    ADD CONSTRAINT links_pkey PRIMARY KEY (id);


--
-- Name: logs logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY logs
    ADD CONSTRAINT logs_pkey PRIMARY KEY (id);


--
-- Name: nodes nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY nodes
    ADD CONSTRAINT nodes_pkey PRIMARY KEY (id);


--
-- Name: permission_refresh_lock permission_refresh_lock_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY permission_refresh_lock
    ADD CONSTRAINT permission_refresh_lock_pkey PRIMARY KEY (id);


--
-- Name: pipeline_instances pipeline_instances_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY pipeline_instances
    ADD CONSTRAINT pipeline_instances_pkey PRIMARY KEY (id);


--
-- Name: pipeline_templates pipeline_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY pipeline_templates
    ADD CONSTRAINT pipeline_templates_pkey PRIMARY KEY (id);


--
-- Name: repositories repositories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY repositories
    ADD CONSTRAINT repositories_pkey PRIMARY KEY (id);


--
-- Name: specimens specimens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY specimens
    ADD CONSTRAINT specimens_pkey PRIMARY KEY (id);


--
-- Name: traits traits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY traits
    ADD CONSTRAINT traits_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: virtual_machines virtual_machines_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY virtual_machines
    ADD CONSTRAINT virtual_machines_pkey PRIMARY KEY (id);


--
-- Name: workflows workflows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY workflows
    ADD CONSTRAINT workflows_pkey PRIMARY KEY (id);


--
-- Name: api_client_authorizations_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX api_client_authorizations_search_index ON api_client_authorizations USING btree (api_token, created_by_ip_address, last_used_by_ip_address, default_owner_uuid, uuid);


--
-- Name: api_clients_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX api_clients_search_index ON api_clients USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name, url_prefix);


--
-- Name: authorized_keys_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX authorized_keys_search_index ON authorized_keys USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name, key_type, authorized_user_uuid);


--
-- Name: collection_index_on_properties; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX collection_index_on_properties ON collections USING gin (properties);


--
-- Name: collections_full_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX collections_full_text_search_idx ON collections USING gin (to_tsvector('english'::regconfig, (((((((((((((((((COALESCE(owner_uuid, ''::character varying))::text || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(portable_data_hash, ''::character varying))::text) || ' '::text) || (COALESCE(uuid, ''::character varying))::text) || ' '::text) || (COALESCE(name, ''::character varying))::text) || ' '::text) || (COALESCE(description, ''::character varying))::text) || ' '::text) || COALESCE((properties)::text, ''::text)) || ' '::text) || (COALESCE(file_names, ''::character varying))::text)));


--
-- Name: collections_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX collections_search_index ON collections USING btree (owner_uuid, modified_by_client_uuid, modified_by_user_uuid, portable_data_hash, uuid, name);


--
-- Name: container_requests_full_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX container_requests_full_text_search_idx ON container_requests USING gin (to_tsvector('english'::regconfig, (((((((((((((((((((((((((((((((((((((((((COALESCE(uuid, ''::character varying))::text || ' '::text) || (COALESCE(owner_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(name, ''::character varying))::text) || ' '::text) || COALESCE(description, ''::text)) || ' '::text) || COALESCE((properties)::text, ''::text)) || ' '::text) || (COALESCE(state, ''::character varying))::text) || ' '::text) || (COALESCE(requesting_container_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(container_uuid, ''::character varying))::text) || ' '::text) || COALESCE(runtime_constraints, ''::text)) || ' '::text) || (COALESCE(container_image, ''::character varying))::text) || ' '::text) || COALESCE(environment, ''::text)) || ' '::text) || (COALESCE(cwd, ''::character varying))::text) || ' '::text) || COALESCE(command, ''::text)) || ' '::text) || (COALESCE(output_path, ''::character varying))::text) || ' '::text) || COALESCE(filters, ''::text)) || ' '::text) || COALESCE(scheduling_parameters, ''::text)) || ' '::text) || (COALESCE(output_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(log_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(output_name, ''::character varying))::text)));


--
-- Name: container_requests_index_on_properties; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX container_requests_index_on_properties ON container_requests USING gin (properties);


--
-- Name: container_requests_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX container_requests_search_index ON container_requests USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name, state, requesting_container_uuid, container_uuid, container_image, cwd, output_path, output_uuid, log_uuid, output_name);


--
-- Name: containers_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX containers_search_index ON containers USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, state, log, cwd, output_path, output, container_image, auth_uuid, locked_by_uuid);


--
-- Name: group_index_on_properties; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX group_index_on_properties ON groups USING gin (properties);


--
-- Name: groups_full_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX groups_full_text_search_idx ON groups USING gin (to_tsvector('english'::regconfig, (((((((((((((((COALESCE(uuid, ''::character varying))::text || ' '::text) || (COALESCE(owner_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(name, ''::character varying))::text) || ' '::text) || (COALESCE(description, ''::character varying))::text) || ' '::text) || (COALESCE(group_class, ''::character varying))::text) || ' '::text) || COALESCE((properties)::text, ''::text))));


--
-- Name: groups_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX groups_search_index ON groups USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name, group_class);


--
-- Name: humans_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX humans_search_index ON humans USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid);


--
-- Name: index_api_client_authorizations_on_api_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_api_client_authorizations_on_api_client_id ON api_client_authorizations USING btree (api_client_id);


--
-- Name: index_api_client_authorizations_on_api_token; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_api_client_authorizations_on_api_token ON api_client_authorizations USING btree (api_token);


--
-- Name: index_api_client_authorizations_on_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_api_client_authorizations_on_expires_at ON api_client_authorizations USING btree (expires_at);


--
-- Name: index_api_client_authorizations_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_api_client_authorizations_on_user_id ON api_client_authorizations USING btree (user_id);


--
-- Name: index_api_client_authorizations_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_api_client_authorizations_on_uuid ON api_client_authorizations USING btree (uuid);


--
-- Name: index_api_clients_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_api_clients_on_created_at ON api_clients USING btree (created_at);


--
-- Name: index_api_clients_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_api_clients_on_modified_at ON api_clients USING btree (modified_at);


--
-- Name: index_api_clients_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_api_clients_on_owner_uuid ON api_clients USING btree (owner_uuid);


--
-- Name: index_api_clients_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_api_clients_on_uuid ON api_clients USING btree (uuid);


--
-- Name: index_authkeys_on_user_and_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_authkeys_on_user_and_expires_at ON authorized_keys USING btree (authorized_user_uuid, expires_at);


--
-- Name: index_authorized_keys_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_authorized_keys_on_owner_uuid ON authorized_keys USING btree (owner_uuid);


--
-- Name: index_authorized_keys_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_authorized_keys_on_uuid ON authorized_keys USING btree (uuid);


--
-- Name: index_collections_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_created_at ON collections USING btree (created_at);


--
-- Name: index_collections_on_delete_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_delete_at ON collections USING btree (delete_at);


--
-- Name: index_collections_on_is_trashed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_is_trashed ON collections USING btree (is_trashed);


--
-- Name: index_collections_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_modified_at ON collections USING btree (modified_at);


--
-- Name: index_collections_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_modified_at_uuid ON collections USING btree (modified_at DESC, uuid);


--
-- Name: index_collections_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_owner_uuid ON collections USING btree (owner_uuid);


--
-- Name: index_collections_on_owner_uuid_and_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_collections_on_owner_uuid_and_name ON collections USING btree (owner_uuid, name) WHERE (is_trashed = false);


--
-- Name: index_collections_on_portable_data_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_portable_data_hash ON collections USING btree (portable_data_hash);


--
-- Name: index_collections_on_trash_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_trash_at ON collections USING btree (trash_at);


--
-- Name: index_collections_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_collections_on_uuid ON collections USING btree (uuid);


--
-- Name: index_commit_ancestors_on_descendant_and_ancestor; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_commit_ancestors_on_descendant_and_ancestor ON commit_ancestors USING btree (descendant, ancestor);


--
-- Name: index_commits_on_repository_name_and_sha1; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_commits_on_repository_name_and_sha1 ON commits USING btree (repository_name, sha1);


--
-- Name: index_container_requests_on_container_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_container_requests_on_container_uuid ON container_requests USING btree (container_uuid);


--
-- Name: index_container_requests_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_container_requests_on_modified_at_uuid ON container_requests USING btree (modified_at DESC, uuid);


--
-- Name: index_container_requests_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_container_requests_on_owner_uuid ON container_requests USING btree (owner_uuid);


--
-- Name: index_container_requests_on_requesting_container_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_container_requests_on_requesting_container_uuid ON container_requests USING btree (requesting_container_uuid);


--
-- Name: index_container_requests_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_container_requests_on_uuid ON container_requests USING btree (uuid);


--
-- Name: index_containers_on_auth_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_auth_uuid ON containers USING btree (auth_uuid);


--
-- Name: index_containers_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_modified_at_uuid ON containers USING btree (modified_at DESC, uuid);


--
-- Name: index_containers_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_owner_uuid ON containers USING btree (owner_uuid);


--
-- Name: index_containers_on_secret_mounts_md5; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_secret_mounts_md5 ON containers USING btree (secret_mounts_md5);


--
-- Name: index_containers_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_containers_on_uuid ON containers USING btree (uuid);


--
-- Name: index_groups_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_created_at ON groups USING btree (created_at);


--
-- Name: index_groups_on_delete_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_delete_at ON groups USING btree (delete_at);


--
-- Name: index_groups_on_group_class; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_group_class ON groups USING btree (group_class);


--
-- Name: index_groups_on_is_trashed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_is_trashed ON groups USING btree (is_trashed);


--
-- Name: index_groups_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_modified_at ON groups USING btree (modified_at);


--
-- Name: index_groups_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_modified_at_uuid ON groups USING btree (modified_at DESC, uuid);


--
-- Name: index_groups_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_owner_uuid ON groups USING btree (owner_uuid);


--
-- Name: index_groups_on_owner_uuid_and_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_groups_on_owner_uuid_and_name ON groups USING btree (owner_uuid, name) WHERE (is_trashed = false);


--
-- Name: index_groups_on_trash_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_trash_at ON groups USING btree (trash_at);


--
-- Name: index_groups_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_groups_on_uuid ON groups USING btree (uuid);


--
-- Name: index_humans_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_humans_on_owner_uuid ON humans USING btree (owner_uuid);


--
-- Name: index_humans_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_humans_on_uuid ON humans USING btree (uuid);


--
-- Name: index_job_tasks_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_created_at ON job_tasks USING btree (created_at);


--
-- Name: index_job_tasks_on_created_by_job_task_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_created_by_job_task_uuid ON job_tasks USING btree (created_by_job_task_uuid);


--
-- Name: index_job_tasks_on_job_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_job_uuid ON job_tasks USING btree (job_uuid);


--
-- Name: index_job_tasks_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_modified_at ON job_tasks USING btree (modified_at);


--
-- Name: index_job_tasks_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_owner_uuid ON job_tasks USING btree (owner_uuid);


--
-- Name: index_job_tasks_on_sequence; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_sequence ON job_tasks USING btree (sequence);


--
-- Name: index_job_tasks_on_success; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_success ON job_tasks USING btree (success);


--
-- Name: index_job_tasks_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_job_tasks_on_uuid ON job_tasks USING btree (uuid);


--
-- Name: index_jobs_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_created_at ON jobs USING btree (created_at);


--
-- Name: index_jobs_on_finished_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_finished_at ON jobs USING btree (finished_at);


--
-- Name: index_jobs_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_modified_at ON jobs USING btree (modified_at);


--
-- Name: index_jobs_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_modified_at_uuid ON jobs USING btree (modified_at DESC, uuid);


--
-- Name: index_jobs_on_output; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_output ON jobs USING btree (output);


--
-- Name: index_jobs_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_owner_uuid ON jobs USING btree (owner_uuid);


--
-- Name: index_jobs_on_script; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_script ON jobs USING btree (script);


--
-- Name: index_jobs_on_script_parameters_digest; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_script_parameters_digest ON jobs USING btree (script_parameters_digest);


--
-- Name: index_jobs_on_started_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_started_at ON jobs USING btree (started_at);


--
-- Name: index_jobs_on_submit_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_jobs_on_submit_id ON jobs USING btree (submit_id);


--
-- Name: index_jobs_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_jobs_on_uuid ON jobs USING btree (uuid);


--
-- Name: index_keep_disks_on_filesystem_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_keep_disks_on_filesystem_uuid ON keep_disks USING btree (filesystem_uuid);


--
-- Name: index_keep_disks_on_last_ping_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_keep_disks_on_last_ping_at ON keep_disks USING btree (last_ping_at);


--
-- Name: index_keep_disks_on_node_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_keep_disks_on_node_uuid ON keep_disks USING btree (node_uuid);


--
-- Name: index_keep_disks_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_keep_disks_on_owner_uuid ON keep_disks USING btree (owner_uuid);


--
-- Name: index_keep_disks_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_keep_disks_on_uuid ON keep_disks USING btree (uuid);


--
-- Name: index_keep_services_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_keep_services_on_owner_uuid ON keep_services USING btree (owner_uuid);


--
-- Name: index_keep_services_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_keep_services_on_uuid ON keep_services USING btree (uuid);


--
-- Name: index_links_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_created_at ON links USING btree (created_at);


--
-- Name: index_links_on_head_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_head_uuid ON links USING btree (head_uuid);


--
-- Name: index_links_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_modified_at ON links USING btree (modified_at);


--
-- Name: index_links_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_modified_at_uuid ON links USING btree (modified_at DESC, uuid);


--
-- Name: index_links_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_owner_uuid ON links USING btree (owner_uuid);


--
-- Name: index_links_on_tail_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_tail_uuid ON links USING btree (tail_uuid);


--
-- Name: index_links_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_links_on_uuid ON links USING btree (uuid);


--
-- Name: index_logs_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_created_at ON logs USING btree (created_at);


--
-- Name: index_logs_on_event_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_event_at ON logs USING btree (event_at);


--
-- Name: index_logs_on_event_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_event_type ON logs USING btree (event_type);


--
-- Name: index_logs_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_modified_at ON logs USING btree (modified_at);


--
-- Name: index_logs_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_modified_at_uuid ON logs USING btree (modified_at DESC, uuid);


--
-- Name: index_logs_on_object_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_object_owner_uuid ON logs USING btree (object_owner_uuid);


--
-- Name: index_logs_on_object_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_object_uuid ON logs USING btree (object_uuid);


--
-- Name: index_logs_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_owner_uuid ON logs USING btree (owner_uuid);


--
-- Name: index_logs_on_summary; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_summary ON logs USING btree (summary);


--
-- Name: index_logs_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_logs_on_uuid ON logs USING btree (uuid);


--
-- Name: index_nodes_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_nodes_on_created_at ON nodes USING btree (created_at);


--
-- Name: index_nodes_on_hostname; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_nodes_on_hostname ON nodes USING btree (hostname);


--
-- Name: index_nodes_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_nodes_on_modified_at ON nodes USING btree (modified_at);


--
-- Name: index_nodes_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_nodes_on_owner_uuid ON nodes USING btree (owner_uuid);


--
-- Name: index_nodes_on_slot_number; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_nodes_on_slot_number ON nodes USING btree (slot_number);


--
-- Name: index_nodes_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_nodes_on_uuid ON nodes USING btree (uuid);


--
-- Name: index_pipeline_instances_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_instances_on_created_at ON pipeline_instances USING btree (created_at);


--
-- Name: index_pipeline_instances_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_instances_on_modified_at ON pipeline_instances USING btree (modified_at);


--
-- Name: index_pipeline_instances_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_instances_on_modified_at_uuid ON pipeline_instances USING btree (modified_at DESC, uuid);


--
-- Name: index_pipeline_instances_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_instances_on_owner_uuid ON pipeline_instances USING btree (owner_uuid);


--
-- Name: index_pipeline_instances_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_pipeline_instances_on_uuid ON pipeline_instances USING btree (uuid);


--
-- Name: index_pipeline_templates_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_templates_on_created_at ON pipeline_templates USING btree (created_at);


--
-- Name: index_pipeline_templates_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_templates_on_modified_at ON pipeline_templates USING btree (modified_at);


--
-- Name: index_pipeline_templates_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_templates_on_modified_at_uuid ON pipeline_templates USING btree (modified_at DESC, uuid);


--
-- Name: index_pipeline_templates_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_templates_on_owner_uuid ON pipeline_templates USING btree (owner_uuid);


--
-- Name: index_pipeline_templates_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_pipeline_templates_on_uuid ON pipeline_templates USING btree (uuid);


--
-- Name: index_repositories_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_repositories_on_modified_at_uuid ON repositories USING btree (modified_at DESC, uuid);


--
-- Name: index_repositories_on_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_repositories_on_name ON repositories USING btree (name);


--
-- Name: index_repositories_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_repositories_on_owner_uuid ON repositories USING btree (owner_uuid);


--
-- Name: index_repositories_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_repositories_on_uuid ON repositories USING btree (uuid);


--
-- Name: index_specimens_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_specimens_on_created_at ON specimens USING btree (created_at);


--
-- Name: index_specimens_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_specimens_on_modified_at ON specimens USING btree (modified_at);


--
-- Name: index_specimens_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_specimens_on_owner_uuid ON specimens USING btree (owner_uuid);


--
-- Name: index_specimens_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_specimens_on_uuid ON specimens USING btree (uuid);


--
-- Name: index_traits_on_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_traits_on_name ON traits USING btree (name);


--
-- Name: index_traits_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_traits_on_owner_uuid ON traits USING btree (owner_uuid);


--
-- Name: index_traits_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_traits_on_uuid ON traits USING btree (uuid);


--
-- Name: index_users_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_created_at ON users USING btree (created_at);


--
-- Name: index_users_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_modified_at ON users USING btree (modified_at);


--
-- Name: index_users_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_modified_at_uuid ON users USING btree (modified_at DESC, uuid);


--
-- Name: index_users_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_owner_uuid ON users USING btree (owner_uuid);


--
-- Name: index_users_on_username; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_username ON users USING btree (username);


--
-- Name: index_users_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_uuid ON users USING btree (uuid);


--
-- Name: index_virtual_machines_on_hostname; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_virtual_machines_on_hostname ON virtual_machines USING btree (hostname);


--
-- Name: index_virtual_machines_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_virtual_machines_on_modified_at_uuid ON virtual_machines USING btree (modified_at DESC, uuid);


--
-- Name: index_virtual_machines_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_virtual_machines_on_owner_uuid ON virtual_machines USING btree (owner_uuid);


--
-- Name: index_virtual_machines_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_virtual_machines_on_uuid ON virtual_machines USING btree (uuid);


--
-- Name: index_workflows_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_workflows_on_modified_at_uuid ON workflows USING btree (modified_at DESC, uuid);


--
-- Name: index_workflows_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_workflows_on_owner_uuid ON workflows USING btree (owner_uuid);


--
-- Name: index_workflows_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_workflows_on_uuid ON workflows USING btree (uuid);


--
-- Name: job_tasks_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX job_tasks_search_index ON job_tasks USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, job_uuid, created_by_job_task_uuid);


--
-- Name: jobs_full_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX jobs_full_text_search_idx ON jobs USING gin (to_tsvector('english'::regconfig, (((((((((((((((((((((((((((((((((((((((((((COALESCE(uuid, ''::character varying))::text || ' '::text) || (COALESCE(owner_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(submit_id, ''::character varying))::text) || ' '::text) || (COALESCE(script, ''::character varying))::text) || ' '::text) || (COALESCE(script_version, ''::character varying))::text) || ' '::text) || COALESCE(script_parameters, ''::text)) || ' '::text) || (COALESCE(cancelled_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(cancelled_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(output, ''::character varying))::text) || ' '::text) || (COALESCE(is_locked_by_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(log, ''::character varying))::text) || ' '::text) || COALESCE(tasks_summary, ''::text)) || ' '::text) || COALESCE(runtime_constraints, ''::text)) || ' '::text) || (COALESCE(repository, ''::character varying))::text) || ' '::text) || (COALESCE(supplied_script_version, ''::character varying))::text) || ' '::text) || (COALESCE(docker_image_locator, ''::character varying))::text) || ' '::text) || (COALESCE(description, ''::character varying))::text) || ' '::text) || (COALESCE(state, ''::character varying))::text) || ' '::text) || (COALESCE(arvados_sdk_version, ''::character varying))::text) || ' '::text) || COALESCE(components, ''::text))));


--
-- Name: jobs_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX jobs_search_index ON jobs USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, submit_id, script, script_version, cancelled_by_client_uuid, cancelled_by_user_uuid, output, is_locked_by_uuid, log, repository, supplied_script_version, docker_image_locator, state, arvados_sdk_version);


--
-- Name: keep_disks_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX keep_disks_search_index ON keep_disks USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, ping_secret, node_uuid, filesystem_uuid, keep_service_uuid);


--
-- Name: keep_services_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX keep_services_search_index ON keep_services USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, service_host, service_type);


--
-- Name: links_index_on_properties; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX links_index_on_properties ON links USING gin (properties);


--
-- Name: links_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX links_search_index ON links USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, tail_uuid, link_class, name, head_uuid);


--
-- Name: links_tail_name_unique_if_link_class_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX links_tail_name_unique_if_link_class_name ON links USING btree (tail_uuid, name) WHERE ((link_class)::text = 'name'::text);


--
-- Name: logs_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX logs_search_index ON logs USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, object_uuid, event_type, object_owner_uuid);


--
-- Name: nodes_index_on_info; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX nodes_index_on_info ON nodes USING gin (info);


--
-- Name: nodes_index_on_properties; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX nodes_index_on_properties ON nodes USING gin (properties);


--
-- Name: nodes_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX nodes_search_index ON nodes USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, hostname, domain, ip_address, job_uuid);


--
-- Name: permission_target_trashed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX permission_target_trashed ON materialized_permission_view USING btree (trashed, target_uuid);


--
-- Name: permission_target_user_trashed_level; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX permission_target_user_trashed_level ON materialized_permission_view USING btree (user_uuid, trashed, perm_level);


--
-- Name: pipeline_instances_full_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pipeline_instances_full_text_search_idx ON pipeline_instances USING gin (to_tsvector('english'::regconfig, (((((((((((((((((((((COALESCE(uuid, ''::character varying))::text || ' '::text) || (COALESCE(owner_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(pipeline_template_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(name, ''::character varying))::text) || ' '::text) || COALESCE(components, ''::text)) || ' '::text) || COALESCE(properties, ''::text)) || ' '::text) || (COALESCE(state, ''::character varying))::text) || ' '::text) || COALESCE(components_summary, ''::text)) || ' '::text) || (COALESCE(description, ''::character varying))::text)));


--
-- Name: pipeline_instances_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pipeline_instances_search_index ON pipeline_instances USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, pipeline_template_uuid, name, state);


--
-- Name: pipeline_template_owner_uuid_name_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX pipeline_template_owner_uuid_name_unique ON pipeline_templates USING btree (owner_uuid, name);


--
-- Name: pipeline_templates_full_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pipeline_templates_full_text_search_idx ON pipeline_templates USING gin (to_tsvector('english'::regconfig, (((((((((((((COALESCE(uuid, ''::character varying))::text || ' '::text) || (COALESCE(owner_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(name, ''::character varying))::text) || ' '::text) || COALESCE(components, ''::text)) || ' '::text) || (COALESCE(description, ''::character varying))::text)));


--
-- Name: pipeline_templates_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pipeline_templates_search_index ON pipeline_templates USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name);


--
-- Name: repositories_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repositories_search_index ON repositories USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name);


--
-- Name: specimens_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX specimens_search_index ON specimens USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, material);


--
-- Name: traits_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX traits_search_index ON traits USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name);


--
-- Name: unique_schema_migrations; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX unique_schema_migrations ON schema_migrations USING btree (version);


--
-- Name: users_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX users_search_index ON users USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, email, first_name, last_name, identity_url, default_owner_uuid, username, redirect_to_user_uuid);


--
-- Name: virtual_machines_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX virtual_machines_search_index ON virtual_machines USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, hostname);


--
-- Name: workflows_full_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workflows_full_text_search_idx ON workflows USING gin (to_tsvector('english'::regconfig, (((((((((((COALESCE(uuid, ''::character varying))::text || ' '::text) || (COALESCE(owner_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(name, ''::character varying))::text) || ' '::text) || COALESCE(description, ''::text))));


--
-- Name: workflows_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workflows_search_idx ON workflows USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name);


--
-- PostgreSQL database dump complete
--

SET search_path TO "$user", public;

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

INSERT INTO schema_migrations (version) VALUES ('20140817035914');

INSERT INTO schema_migrations (version) VALUES ('20140818125735');

INSERT INTO schema_migrations (version) VALUES ('20140826180337');

INSERT INTO schema_migrations (version) VALUES ('20140828141043');

INSERT INTO schema_migrations (version) VALUES ('20140909183946');

INSERT INTO schema_migrations (version) VALUES ('20140911221252');

INSERT INTO schema_migrations (version) VALUES ('20140918141529');

INSERT INTO schema_migrations (version) VALUES ('20140918153541');

INSERT INTO schema_migrations (version) VALUES ('20140918153705');

INSERT INTO schema_migrations (version) VALUES ('20140924091559');

INSERT INTO schema_migrations (version) VALUES ('20141111133038');

INSERT INTO schema_migrations (version) VALUES ('20141208164553');

INSERT INTO schema_migrations (version) VALUES ('20141208174553');

INSERT INTO schema_migrations (version) VALUES ('20141208174653');

INSERT INTO schema_migrations (version) VALUES ('20141208185217');

INSERT INTO schema_migrations (version) VALUES ('20150122175935');

INSERT INTO schema_migrations (version) VALUES ('20150123142953');

INSERT INTO schema_migrations (version) VALUES ('20150203180223');

INSERT INTO schema_migrations (version) VALUES ('20150206210804');

INSERT INTO schema_migrations (version) VALUES ('20150206230342');

INSERT INTO schema_migrations (version) VALUES ('20150216193428');

INSERT INTO schema_migrations (version) VALUES ('20150303210106');

INSERT INTO schema_migrations (version) VALUES ('20150312151136');

INSERT INTO schema_migrations (version) VALUES ('20150317132720');

INSERT INTO schema_migrations (version) VALUES ('20150324152204');

INSERT INTO schema_migrations (version) VALUES ('20150423145759');

INSERT INTO schema_migrations (version) VALUES ('20150512193020');

INSERT INTO schema_migrations (version) VALUES ('20150526180251');

INSERT INTO schema_migrations (version) VALUES ('20151202151426');

INSERT INTO schema_migrations (version) VALUES ('20151215134304');

INSERT INTO schema_migrations (version) VALUES ('20151229214707');

INSERT INTO schema_migrations (version) VALUES ('20160208210629');

INSERT INTO schema_migrations (version) VALUES ('20160209155729');

INSERT INTO schema_migrations (version) VALUES ('20160324144017');

INSERT INTO schema_migrations (version) VALUES ('20160506175108');

INSERT INTO schema_migrations (version) VALUES ('20160509143250');

INSERT INTO schema_migrations (version) VALUES ('20160808151559');

INSERT INTO schema_migrations (version) VALUES ('20160819195557');

INSERT INTO schema_migrations (version) VALUES ('20160819195725');

INSERT INTO schema_migrations (version) VALUES ('20160901210110');

INSERT INTO schema_migrations (version) VALUES ('20160909181442');

INSERT INTO schema_migrations (version) VALUES ('20160926194129');

INSERT INTO schema_migrations (version) VALUES ('20161019171346');

INSERT INTO schema_migrations (version) VALUES ('20161111143147');

INSERT INTO schema_migrations (version) VALUES ('20161115171221');

INSERT INTO schema_migrations (version) VALUES ('20161115174218');

INSERT INTO schema_migrations (version) VALUES ('20161213172944');

INSERT INTO schema_migrations (version) VALUES ('20161222153434');

INSERT INTO schema_migrations (version) VALUES ('20161223090712');

INSERT INTO schema_migrations (version) VALUES ('20170102153111');

INSERT INTO schema_migrations (version) VALUES ('20170105160301');

INSERT INTO schema_migrations (version) VALUES ('20170105160302');

INSERT INTO schema_migrations (version) VALUES ('20170216170823');

INSERT INTO schema_migrations (version) VALUES ('20170301225558');

INSERT INTO schema_migrations (version) VALUES ('20170319063406');

INSERT INTO schema_migrations (version) VALUES ('20170328215436');

INSERT INTO schema_migrations (version) VALUES ('20170330012505');

INSERT INTO schema_migrations (version) VALUES ('20170419173031');

INSERT INTO schema_migrations (version) VALUES ('20170419173712');

INSERT INTO schema_migrations (version) VALUES ('20170419175801');

INSERT INTO schema_migrations (version) VALUES ('20170628185847');

INSERT INTO schema_migrations (version) VALUES ('20170704160233');

INSERT INTO schema_migrations (version) VALUES ('20170706141334');

INSERT INTO schema_migrations (version) VALUES ('20170824202826');

INSERT INTO schema_migrations (version) VALUES ('20170906224040');

INSERT INTO schema_migrations (version) VALUES ('20171027183824');

INSERT INTO schema_migrations (version) VALUES ('20171208203841');

INSERT INTO schema_migrations (version) VALUES ('20171212153352');

INSERT INTO schema_migrations (version) VALUES ('20180216203422');

INSERT INTO schema_migrations (version) VALUES ('20180228220311');

INSERT INTO schema_migrations (version) VALUES ('20180313180114');

INSERT INTO schema_migrations (version) VALUES ('20180501182859');

INSERT INTO schema_migrations (version) VALUES ('20180514135529');

INSERT INTO schema_migrations (version) VALUES ('20180608123145');

INSERT INTO schema_migrations (version) VALUES ('20180607175050');
