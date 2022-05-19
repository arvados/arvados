-- Copyright (C) The Arvados Authors. All rights reserved.
--
-- SPDX-License-Identifier: AGPL-3.0

SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;

--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: -
--

-- COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


--
-- Name: compute_permission_subgraph(character varying, character varying, integer, character varying); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.compute_permission_subgraph(perm_origin_uuid character varying, starting_uuid character varying, starting_perm integer, perm_edge_id character varying) RETURNS TABLE(user_uuid character varying, target_uuid character varying, val integer, traverse_owned boolean)
    LANGUAGE sql STABLE
    AS $$

/* The purpose of this function is to compute the permissions for a
   subgraph of the database, starting from a given edge.  The newly
   computed permissions are used to add and remove rows from the main
   permissions table.

   perm_origin_uuid: The object that 'gets' the permission.

   starting_uuid: The starting object the permission applies to.

   starting_perm: The permission that perm_origin_uuid 'has' on
                  starting_uuid One of 1, 2, 3 for can_read,
                  can_write, can_manage respectively, or 0 to revoke
                  permissions.

   perm_edge_id: Identifies the permission edge that is being updated.
                 Changes of ownership, this is starting_uuid.
                 For links, this is the uuid of the link object.
                 This is used to override the edge value in the database
                 with starting_perm.  This is necessary when revoking
                 permissions because the update happens before edge is
                 actually removed.
*/
with
  /* Starting from starting_uuid, determine the set of objects that
     could be affected by this permission change.

     Note: We don't traverse users unless it is an "identity"
     permission (permission origin is self).
  */
  perm_from_start(perm_origin_uuid, target_uuid, val, traverse_owned) as (
    
WITH RECURSIVE
        traverse_graph(origin_uuid, target_uuid, val, traverse_owned, starting_set) as (
            
             values (perm_origin_uuid, starting_uuid, starting_perm,
                    should_traverse_owned(starting_uuid, starting_perm),
                    (perm_origin_uuid = starting_uuid or starting_uuid not like '_____-tpzed-_______________'))

          union
            (select traverse_graph.origin_uuid,
                    edges.head_uuid,
                      least(
case (edges.edge_id = perm_edge_id)
                               when true then starting_perm
                               else edges.val
                            end
,
                            traverse_graph.val),
                    should_traverse_owned(edges.head_uuid, edges.val),
                    false
             from permission_graph_edges as edges, traverse_graph
             where traverse_graph.target_uuid = edges.tail_uuid
             and (edges.tail_uuid like '_____-j7d0g-_______________' or
                  traverse_graph.starting_set)))
        select traverse_graph.origin_uuid, target_uuid, max(val) as val, bool_or(traverse_owned) as traverse_owned from traverse_graph
        group by (traverse_graph.origin_uuid, target_uuid)
),

  /* Find other inbound edges that grant permissions to 'targets' in
     perm_from_start, and compute permissions that originate from
     those.

     This is necessary for two reasons:

       1) Other users may have access to a subset of the objects
       through other permission links than the one we started from.
       If we don't recompute them, their permission will get dropped.

       2) There may be more than one path through which a user gets
       permission to an object.  For example, a user owns a project
       and also shares it can_read with a group the user belongs
       to. adding the can_read link must not overwrite the existing
       can_manage permission granted by ownership.
  */
  additional_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
    
WITH RECURSIVE
        traverse_graph(origin_uuid, target_uuid, val, traverse_owned, starting_set) as (
            
    select edges.tail_uuid as origin_uuid, edges.head_uuid as target_uuid, edges.val,
           should_traverse_owned(edges.head_uuid, edges.val),
           edges.head_uuid like '_____-j7d0g-_______________'
      from permission_graph_edges as edges
      where edges.edge_id != perm_edge_id and
            edges.tail_uuid not in (select target_uuid from perm_from_start where target_uuid like '_____-j7d0g-_______________') and
            edges.head_uuid in (select target_uuid from perm_from_start)

          union
            (select traverse_graph.origin_uuid,
                    edges.head_uuid,
                      least(
case (edges.edge_id = perm_edge_id)
                               when true then starting_perm
                               else edges.val
                            end
,
                            traverse_graph.val),
                    should_traverse_owned(edges.head_uuid, edges.val),
                    false
             from permission_graph_edges as edges, traverse_graph
             where traverse_graph.target_uuid = edges.tail_uuid
             and (edges.tail_uuid like '_____-j7d0g-_______________' or
                  traverse_graph.starting_set)))
        select traverse_graph.origin_uuid, target_uuid, max(val) as val, bool_or(traverse_owned) as traverse_owned from traverse_graph
        group by (traverse_graph.origin_uuid, target_uuid)
),

  /* Combine the permissions computed in the first two phases. */
  all_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
      select * from perm_from_start
    union all
      select * from additional_perms
  )

  /* The actual query that produces rows to be added or removed
     from the materialized_permissions table.  This is the clever
     bit.

     Key insights:

     * For every group, the materialized_permissions lists all users
       that can access to that group.

     * The all_perms subquery has computed permissions on on a set of
       objects for all inbound "origins", which are users or groups.

     * Permissions through groups are transitive.

     We can infer:

     1) The materialized_permissions table declares that user X has permission N on group Y
     2) The all_perms result has determined group Y has permission M on object Z
     3) Therefore, user X has permission min(N, M) on object Z

     This allows us to efficiently determine the set of users that
     have permissions on the subset of objects, without having to
     follow the chain of permission back up to find those users.

     In addition, because users always have permission on themselves, this
     query also makes sure those permission rows are always
     returned.
  */
  select v.user_uuid, v.target_uuid, max(v.perm_level), bool_or(v.traverse_owned) from
    (select m.user_uuid,
         u.target_uuid,
         least(u.val, m.perm_level) as perm_level,
         u.traverse_owned
      from all_perms as u, materialized_permissions as m
           where u.perm_origin_uuid = m.target_uuid AND m.traverse_owned
           AND (m.user_uuid = m.target_uuid or m.target_uuid not like '_____-tpzed-_______________')
    union all
      select target_uuid as user_uuid, target_uuid, 3, true
        from all_perms
        where all_perms.target_uuid like '_____-tpzed-_______________') as v
    group by v.user_uuid, v.target_uuid
$$;


--
-- Name: project_subtree_with_is_frozen(character varying, boolean); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.project_subtree_with_is_frozen(starting_uuid character varying, starting_is_frozen boolean) RETURNS TABLE(uuid character varying, is_frozen boolean)
    LANGUAGE sql STABLE
    AS $$
WITH RECURSIVE
  project_subtree(uuid, is_frozen) as (
    values (starting_uuid, starting_is_frozen)
    union
    select groups.uuid, project_subtree.is_frozen or groups.frozen_by_uuid is not null
      from groups join project_subtree on (groups.owner_uuid = project_subtree.uuid)
  )
  select uuid, is_frozen from project_subtree;
$$;


--
-- Name: project_subtree_with_trash_at(character varying, timestamp without time zone); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.project_subtree_with_trash_at(starting_uuid character varying, starting_trash_at timestamp without time zone) RETURNS TABLE(target_uuid character varying, trash_at timestamp without time zone)
    LANGUAGE sql STABLE
    AS $$
/* Starting from a project, recursively traverse all the projects
  underneath it and return a set of project uuids and trash_at times
  (may be null).  The initial trash_at can be a timestamp or null.
  The trash_at time propagates downward to groups it owns, i.e. when a
  group is trashed, everything underneath it in the ownership
  hierarchy is also considered trashed.  However, this is fact is
  recorded in the trashed_groups table, not by updating trash_at field
  in the groups table.
*/
WITH RECURSIVE
        project_subtree(uuid, trash_at) as (
        values (starting_uuid, starting_trash_at)
        union
        select groups.uuid, LEAST(project_subtree.trash_at, groups.trash_at)
          from groups join project_subtree on (groups.owner_uuid = project_subtree.uuid)
        )
        select uuid, trash_at from project_subtree;
$$;


--
-- Name: should_traverse_owned(character varying, integer); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.should_traverse_owned(starting_uuid character varying, starting_perm integer) RETURNS boolean
    LANGUAGE sql IMMUTABLE
    AS $$
/* Helper function.  Determines if permission on an object implies
   transitive permission to things the object owns.  This is always
   true for groups, but only true for users when the permission level
   is can_manage.
*/
select starting_uuid like '_____-j7d0g-_______________' or
       (starting_uuid like '_____-tpzed-_______________' and starting_perm >= 3);
$$;


SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: api_client_authorizations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_client_authorizations (
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

CREATE SEQUENCE public.api_client_authorizations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: api_client_authorizations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.api_client_authorizations_id_seq OWNED BY public.api_client_authorizations.id;


--
-- Name: api_clients; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_clients (
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

CREATE SEQUENCE public.api_clients_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: api_clients_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.api_clients_id_seq OWNED BY public.api_clients.id;


--
-- Name: ar_internal_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ar_internal_metadata (
    key character varying NOT NULL,
    value character varying,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: authorized_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.authorized_keys (
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

CREATE SEQUENCE public.authorized_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: authorized_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.authorized_keys_id_seq OWNED BY public.authorized_keys.id;


--
-- Name: collections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.collections (
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
    file_names text,
    trash_at timestamp without time zone,
    is_trashed boolean DEFAULT false NOT NULL,
    storage_classes_desired jsonb DEFAULT '["default"]'::jsonb,
    storage_classes_confirmed jsonb DEFAULT '[]'::jsonb,
    storage_classes_confirmed_at timestamp without time zone,
    current_version_uuid character varying,
    version integer DEFAULT 1 NOT NULL,
    preserve_version boolean DEFAULT false,
    file_count integer DEFAULT 0 NOT NULL,
    file_size_total bigint DEFAULT 0 NOT NULL
);


--
-- Name: collections_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.collections_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: collections_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.collections_id_seq OWNED BY public.collections.id;


--
-- Name: container_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.container_requests (
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
    secret_mounts jsonb DEFAULT '{}'::jsonb,
    runtime_token text,
    output_storage_classes jsonb DEFAULT '["default"]'::jsonb,
    output_properties jsonb DEFAULT '{}'::jsonb
);


--
-- Name: container_requests_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.container_requests_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: container_requests_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.container_requests_id_seq OWNED BY public.container_requests.id;


--
-- Name: containers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.containers (
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
    secret_mounts_md5 character varying DEFAULT '99914b932bd37a50b983c5e7c90ae93b'::character varying,
    runtime_status jsonb DEFAULT '{}'::jsonb,
    runtime_user_uuid text,
    runtime_auth_scopes jsonb,
    runtime_token text,
    lock_count integer DEFAULT 0 NOT NULL,
    gateway_address character varying,
    interactive_session_started boolean DEFAULT false NOT NULL,
    output_storage_classes jsonb DEFAULT '["default"]'::jsonb,
    output_properties jsonb DEFAULT '{}'::jsonb
);


--
-- Name: containers_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.containers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: containers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.containers_id_seq OWNED BY public.containers.id;


--
-- Name: frozen_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.frozen_groups (
    uuid character varying
);


--
-- Name: groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.groups (
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
    properties jsonb DEFAULT '{}'::jsonb,
    frozen_by_uuid character varying
);


--
-- Name: groups_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: groups_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.groups_id_seq OWNED BY public.groups.id;


--
-- Name: humans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.humans (
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

CREATE SEQUENCE public.humans_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: humans_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.humans_id_seq OWNED BY public.humans.id;


--
-- Name: job_tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.job_tasks (
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

CREATE SEQUENCE public.job_tasks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: job_tasks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.job_tasks_id_seq OWNED BY public.job_tasks.id;


--
-- Name: job_tasks_qsequence_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.job_tasks_qsequence_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: job_tasks_qsequence_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.job_tasks_qsequence_seq OWNED BY public.job_tasks.qsequence;


--
-- Name: jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.jobs (
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

CREATE SEQUENCE public.jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.jobs_id_seq OWNED BY public.jobs.id;


--
-- Name: keep_disks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.keep_disks (
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

CREATE SEQUENCE public.keep_disks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: keep_disks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.keep_disks_id_seq OWNED BY public.keep_disks.id;


--
-- Name: keep_services; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.keep_services (
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

CREATE SEQUENCE public.keep_services_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: keep_services_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.keep_services_id_seq OWNED BY public.keep_services.id;


--
-- Name: links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.links (
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

CREATE SEQUENCE public.links_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: links_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.links_id_seq OWNED BY public.links.id;


--
-- Name: logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.logs (
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

CREATE SEQUENCE public.logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.logs_id_seq OWNED BY public.logs.id;


--
-- Name: materialized_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.materialized_permissions (
    user_uuid character varying,
    target_uuid character varying,
    perm_level integer,
    traverse_owned boolean
);


--
-- Name: nodes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.nodes (
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

CREATE SEQUENCE public.nodes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: nodes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.nodes_id_seq OWNED BY public.nodes.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
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
-- Name: permission_graph_edges; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.permission_graph_edges AS
 SELECT groups.owner_uuid AS tail_uuid,
    groups.uuid AS head_uuid,
    3 AS val,
    groups.uuid AS edge_id
   FROM public.groups
UNION ALL
 SELECT users.owner_uuid AS tail_uuid,
    users.uuid AS head_uuid,
    3 AS val,
    users.uuid AS edge_id
   FROM public.users
UNION ALL
 SELECT users.uuid AS tail_uuid,
    users.uuid AS head_uuid,
    3 AS val,
    ''::character varying AS edge_id
   FROM public.users
UNION ALL
 SELECT links.tail_uuid,
    links.head_uuid,
        CASE
            WHEN ((links.name)::text = 'can_read'::text) THEN 1
            WHEN ((links.name)::text = 'can_login'::text) THEN 1
            WHEN ((links.name)::text = 'can_write'::text) THEN 2
            WHEN ((links.name)::text = 'can_manage'::text) THEN 3
            ELSE 0
        END AS val,
    links.uuid AS edge_id
   FROM public.links
  WHERE ((links.link_class)::text = 'permission'::text);


--
-- Name: pipeline_instances; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pipeline_instances (
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

CREATE SEQUENCE public.pipeline_instances_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: pipeline_instances_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.pipeline_instances_id_seq OWNED BY public.pipeline_instances.id;


--
-- Name: pipeline_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pipeline_templates (
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

CREATE SEQUENCE public.pipeline_templates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: pipeline_templates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.pipeline_templates_id_seq OWNED BY public.pipeline_templates.id;


--
-- Name: repositories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repositories (
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

CREATE SEQUENCE public.repositories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: repositories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.repositories_id_seq OWNED BY public.repositories.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying(255) NOT NULL
);


--
-- Name: specimens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.specimens (
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

CREATE SEQUENCE public.specimens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: specimens_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.specimens_id_seq OWNED BY public.specimens.id;


--
-- Name: traits; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.traits (
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

CREATE SEQUENCE public.traits_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: traits_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.traits_id_seq OWNED BY public.traits.id;


--
-- Name: trashed_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trashed_groups (
    group_uuid character varying,
    trash_at timestamp without time zone
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: virtual_machines; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.virtual_machines (
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

CREATE SEQUENCE public.virtual_machines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: virtual_machines_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.virtual_machines_id_seq OWNED BY public.virtual_machines.id;


--
-- Name: workflows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflows (
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

CREATE SEQUENCE public.workflows_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: workflows_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.workflows_id_seq OWNED BY public.workflows.id;


--
-- Name: api_client_authorizations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_client_authorizations ALTER COLUMN id SET DEFAULT nextval('public.api_client_authorizations_id_seq'::regclass);


--
-- Name: api_clients id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_clients ALTER COLUMN id SET DEFAULT nextval('public.api_clients_id_seq'::regclass);


--
-- Name: authorized_keys id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.authorized_keys ALTER COLUMN id SET DEFAULT nextval('public.authorized_keys_id_seq'::regclass);


--
-- Name: collections id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.collections ALTER COLUMN id SET DEFAULT nextval('public.collections_id_seq'::regclass);


--
-- Name: container_requests id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.container_requests ALTER COLUMN id SET DEFAULT nextval('public.container_requests_id_seq'::regclass);


--
-- Name: containers id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.containers ALTER COLUMN id SET DEFAULT nextval('public.containers_id_seq'::regclass);


--
-- Name: groups id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.groups ALTER COLUMN id SET DEFAULT nextval('public.groups_id_seq'::regclass);


--
-- Name: humans id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.humans ALTER COLUMN id SET DEFAULT nextval('public.humans_id_seq'::regclass);


--
-- Name: job_tasks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_tasks ALTER COLUMN id SET DEFAULT nextval('public.job_tasks_id_seq'::regclass);


--
-- Name: jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.jobs ALTER COLUMN id SET DEFAULT nextval('public.jobs_id_seq'::regclass);


--
-- Name: keep_disks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.keep_disks ALTER COLUMN id SET DEFAULT nextval('public.keep_disks_id_seq'::regclass);


--
-- Name: keep_services id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.keep_services ALTER COLUMN id SET DEFAULT nextval('public.keep_services_id_seq'::regclass);


--
-- Name: links id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.links ALTER COLUMN id SET DEFAULT nextval('public.links_id_seq'::regclass);


--
-- Name: logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.logs ALTER COLUMN id SET DEFAULT nextval('public.logs_id_seq'::regclass);


--
-- Name: nodes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nodes ALTER COLUMN id SET DEFAULT nextval('public.nodes_id_seq'::regclass);


--
-- Name: pipeline_instances id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline_instances ALTER COLUMN id SET DEFAULT nextval('public.pipeline_instances_id_seq'::regclass);


--
-- Name: pipeline_templates id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline_templates ALTER COLUMN id SET DEFAULT nextval('public.pipeline_templates_id_seq'::regclass);


--
-- Name: repositories id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repositories ALTER COLUMN id SET DEFAULT nextval('public.repositories_id_seq'::regclass);


--
-- Name: specimens id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.specimens ALTER COLUMN id SET DEFAULT nextval('public.specimens_id_seq'::regclass);


--
-- Name: traits id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.traits ALTER COLUMN id SET DEFAULT nextval('public.traits_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: virtual_machines id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.virtual_machines ALTER COLUMN id SET DEFAULT nextval('public.virtual_machines_id_seq'::regclass);


--
-- Name: workflows id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflows ALTER COLUMN id SET DEFAULT nextval('public.workflows_id_seq'::regclass);


--
-- Name: api_client_authorizations api_client_authorizations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_client_authorizations
    ADD CONSTRAINT api_client_authorizations_pkey PRIMARY KEY (id);


--
-- Name: api_clients api_clients_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_clients
    ADD CONSTRAINT api_clients_pkey PRIMARY KEY (id);


--
-- Name: ar_internal_metadata ar_internal_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ar_internal_metadata
    ADD CONSTRAINT ar_internal_metadata_pkey PRIMARY KEY (key);


--
-- Name: authorized_keys authorized_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.authorized_keys
    ADD CONSTRAINT authorized_keys_pkey PRIMARY KEY (id);


--
-- Name: collections collections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.collections
    ADD CONSTRAINT collections_pkey PRIMARY KEY (id);


--
-- Name: container_requests container_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.container_requests
    ADD CONSTRAINT container_requests_pkey PRIMARY KEY (id);


--
-- Name: containers containers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.containers
    ADD CONSTRAINT containers_pkey PRIMARY KEY (id);


--
-- Name: groups groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (id);


--
-- Name: humans humans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.humans
    ADD CONSTRAINT humans_pkey PRIMARY KEY (id);


--
-- Name: job_tasks job_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_tasks
    ADD CONSTRAINT job_tasks_pkey PRIMARY KEY (id);


--
-- Name: jobs jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_pkey PRIMARY KEY (id);


--
-- Name: keep_disks keep_disks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.keep_disks
    ADD CONSTRAINT keep_disks_pkey PRIMARY KEY (id);


--
-- Name: keep_services keep_services_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.keep_services
    ADD CONSTRAINT keep_services_pkey PRIMARY KEY (id);


--
-- Name: links links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.links
    ADD CONSTRAINT links_pkey PRIMARY KEY (id);


--
-- Name: logs logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.logs
    ADD CONSTRAINT logs_pkey PRIMARY KEY (id);


--
-- Name: nodes nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nodes
    ADD CONSTRAINT nodes_pkey PRIMARY KEY (id);


--
-- Name: pipeline_instances pipeline_instances_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline_instances
    ADD CONSTRAINT pipeline_instances_pkey PRIMARY KEY (id);


--
-- Name: pipeline_templates pipeline_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline_templates
    ADD CONSTRAINT pipeline_templates_pkey PRIMARY KEY (id);


--
-- Name: repositories repositories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repositories
    ADD CONSTRAINT repositories_pkey PRIMARY KEY (id);


--
-- Name: specimens specimens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.specimens
    ADD CONSTRAINT specimens_pkey PRIMARY KEY (id);


--
-- Name: traits traits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.traits
    ADD CONSTRAINT traits_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: virtual_machines virtual_machines_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.virtual_machines
    ADD CONSTRAINT virtual_machines_pkey PRIMARY KEY (id);


--
-- Name: workflows workflows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflows
    ADD CONSTRAINT workflows_pkey PRIMARY KEY (id);


--
-- Name: api_client_authorizations_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX api_client_authorizations_search_index ON public.api_client_authorizations USING btree (api_token, created_by_ip_address, last_used_by_ip_address, default_owner_uuid, uuid);


--
-- Name: api_clients_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX api_clients_search_index ON public.api_clients USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name, url_prefix);


--
-- Name: authorized_keys_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX authorized_keys_search_index ON public.authorized_keys USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name, key_type, authorized_user_uuid);


--
-- Name: collection_index_on_properties; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX collection_index_on_properties ON public.collections USING gin (properties);


--
-- Name: collections_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX collections_search_index ON public.collections USING btree (owner_uuid, modified_by_client_uuid, modified_by_user_uuid, portable_data_hash, uuid, name, current_version_uuid);


--
-- Name: collections_trgm_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX collections_trgm_text_search_idx ON public.collections USING gin (((((((((((((((((((COALESCE(owner_uuid, ''::character varying))::text || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(portable_data_hash, ''::character varying))::text) || ' '::text) || (COALESCE(uuid, ''::character varying))::text) || ' '::text) || (COALESCE(name, ''::character varying))::text) || ' '::text) || (COALESCE(description, ''::character varying))::text) || ' '::text) || COALESCE((properties)::text, ''::text)) || ' '::text) || COALESCE(file_names, ''::text))) public.gin_trgm_ops);


--
-- Name: container_requests_index_on_properties; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX container_requests_index_on_properties ON public.container_requests USING gin (properties);


--
-- Name: container_requests_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX container_requests_search_index ON public.container_requests USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name, state, requesting_container_uuid, container_uuid, container_image, cwd, output_path, output_uuid, log_uuid, output_name);


--
-- Name: container_requests_trgm_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX container_requests_trgm_text_search_idx ON public.container_requests USING gin (((((((((((((((((((((((((((((((((((((((((((((COALESCE(uuid, ''::character varying))::text || ' '::text) || (COALESCE(owner_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(name, ''::character varying))::text) || ' '::text) || COALESCE(description, ''::text)) || ' '::text) || COALESCE((properties)::text, ''::text)) || ' '::text) || (COALESCE(state, ''::character varying))::text) || ' '::text) || (COALESCE(requesting_container_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(container_uuid, ''::character varying))::text) || ' '::text) || COALESCE(runtime_constraints, ''::text)) || ' '::text) || (COALESCE(container_image, ''::character varying))::text) || ' '::text) || COALESCE(environment, ''::text)) || ' '::text) || (COALESCE(cwd, ''::character varying))::text) || ' '::text) || COALESCE(command, ''::text)) || ' '::text) || (COALESCE(output_path, ''::character varying))::text) || ' '::text) || COALESCE(filters, ''::text)) || ' '::text) || COALESCE(scheduling_parameters, ''::text)) || ' '::text) || (COALESCE(output_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(log_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(output_name, ''::character varying))::text) || ' '::text) || COALESCE((output_properties)::text, ''::text))) public.gin_trgm_ops);


--
-- Name: containers_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX containers_search_index ON public.containers USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, state, log, cwd, output_path, output, container_image, auth_uuid, locked_by_uuid);


--
-- Name: group_index_on_properties; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX group_index_on_properties ON public.groups USING gin (properties);


--
-- Name: groups_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX groups_search_index ON public.groups USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name, group_class, frozen_by_uuid);


--
-- Name: groups_trgm_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX groups_trgm_text_search_idx ON public.groups USING gin (((((((((((((((((COALESCE(uuid, ''::character varying))::text || ' '::text) || (COALESCE(owner_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(name, ''::character varying))::text) || ' '::text) || (COALESCE(description, ''::character varying))::text) || ' '::text) || (COALESCE(group_class, ''::character varying))::text) || ' '::text) || COALESCE((properties)::text, ''::text))) public.gin_trgm_ops);


--
-- Name: humans_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX humans_search_index ON public.humans USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid);


--
-- Name: index_api_client_authorizations_on_api_client_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_api_client_authorizations_on_api_client_id ON public.api_client_authorizations USING btree (api_client_id);


--
-- Name: index_api_client_authorizations_on_api_token; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_api_client_authorizations_on_api_token ON public.api_client_authorizations USING btree (api_token);


--
-- Name: index_api_client_authorizations_on_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_api_client_authorizations_on_expires_at ON public.api_client_authorizations USING btree (expires_at);


--
-- Name: index_api_client_authorizations_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_api_client_authorizations_on_user_id ON public.api_client_authorizations USING btree (user_id);


--
-- Name: index_api_client_authorizations_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_api_client_authorizations_on_uuid ON public.api_client_authorizations USING btree (uuid);


--
-- Name: index_api_clients_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_api_clients_on_created_at ON public.api_clients USING btree (created_at);


--
-- Name: index_api_clients_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_api_clients_on_modified_at ON public.api_clients USING btree (modified_at);


--
-- Name: index_api_clients_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_api_clients_on_owner_uuid ON public.api_clients USING btree (owner_uuid);


--
-- Name: index_api_clients_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_api_clients_on_uuid ON public.api_clients USING btree (uuid);


--
-- Name: index_authkeys_on_user_and_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_authkeys_on_user_and_expires_at ON public.authorized_keys USING btree (authorized_user_uuid, expires_at);


--
-- Name: index_authorized_keys_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_authorized_keys_on_owner_uuid ON public.authorized_keys USING btree (owner_uuid);


--
-- Name: index_authorized_keys_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_authorized_keys_on_uuid ON public.authorized_keys USING btree (uuid);


--
-- Name: index_collections_on_created_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_created_at_and_uuid ON public.collections USING btree (created_at, uuid);


--
-- Name: index_collections_on_current_version_uuid_and_version; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_collections_on_current_version_uuid_and_version ON public.collections USING btree (current_version_uuid, version);


--
-- Name: index_collections_on_delete_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_delete_at ON public.collections USING btree (delete_at);


--
-- Name: index_collections_on_is_trashed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_is_trashed ON public.collections USING btree (is_trashed);


--
-- Name: index_collections_on_modified_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_modified_at_and_uuid ON public.collections USING btree (modified_at, uuid);


--
-- Name: index_collections_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_owner_uuid ON public.collections USING btree (owner_uuid);


--
-- Name: index_collections_on_owner_uuid_and_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_collections_on_owner_uuid_and_name ON public.collections USING btree (owner_uuid, name) WHERE ((is_trashed = false) AND ((current_version_uuid)::text = (uuid)::text));


--
-- Name: index_collections_on_portable_data_hash_and_trash_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_portable_data_hash_and_trash_at ON public.collections USING btree (portable_data_hash, trash_at);


--
-- Name: index_collections_on_trash_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_collections_on_trash_at ON public.collections USING btree (trash_at);


--
-- Name: index_collections_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_collections_on_uuid ON public.collections USING btree (uuid);


--
-- Name: index_container_requests_on_container_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_container_requests_on_container_uuid ON public.container_requests USING btree (container_uuid);


--
-- Name: index_container_requests_on_created_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_container_requests_on_created_at_and_uuid ON public.container_requests USING btree (created_at, uuid);


--
-- Name: index_container_requests_on_modified_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_container_requests_on_modified_at_and_uuid ON public.container_requests USING btree (modified_at, uuid);


--
-- Name: index_container_requests_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_container_requests_on_owner_uuid ON public.container_requests USING btree (owner_uuid);


--
-- Name: index_container_requests_on_requesting_container_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_container_requests_on_requesting_container_uuid ON public.container_requests USING btree (requesting_container_uuid);


--
-- Name: index_container_requests_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_container_requests_on_uuid ON public.container_requests USING btree (uuid);


--
-- Name: index_containers_on_auth_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_auth_uuid ON public.containers USING btree (auth_uuid);


--
-- Name: index_containers_on_locked_by_uuid_and_priority; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_locked_by_uuid_and_priority ON public.containers USING btree (locked_by_uuid, priority);


--
-- Name: index_containers_on_locked_by_uuid_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_locked_by_uuid_and_uuid ON public.containers USING btree (locked_by_uuid, uuid);


--
-- Name: index_containers_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_modified_at_uuid ON public.containers USING btree (modified_at DESC, uuid);


--
-- Name: index_containers_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_owner_uuid ON public.containers USING btree (owner_uuid);


--
-- Name: index_containers_on_queued_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_queued_state ON public.containers USING btree (state, ((priority > 0)));


--
-- Name: index_containers_on_reuse_columns; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_reuse_columns ON public.containers USING btree (md5(command), cwd, md5(environment), output_path, container_image, md5(mounts), secret_mounts_md5, md5(runtime_constraints));


--
-- Name: index_containers_on_runtime_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_runtime_status ON public.containers USING gin (runtime_status);


--
-- Name: index_containers_on_secret_mounts_md5; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_containers_on_secret_mounts_md5 ON public.containers USING btree (secret_mounts_md5);


--
-- Name: index_containers_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_containers_on_uuid ON public.containers USING btree (uuid);


--
-- Name: index_frozen_groups_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_frozen_groups_on_uuid ON public.frozen_groups USING btree (uuid);


--
-- Name: index_groups_on_created_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_created_at_and_uuid ON public.groups USING btree (created_at, uuid);


--
-- Name: index_groups_on_delete_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_delete_at ON public.groups USING btree (delete_at);


--
-- Name: index_groups_on_group_class; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_group_class ON public.groups USING btree (group_class);


--
-- Name: index_groups_on_is_trashed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_is_trashed ON public.groups USING btree (is_trashed);


--
-- Name: index_groups_on_modified_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_modified_at_and_uuid ON public.groups USING btree (modified_at, uuid);


--
-- Name: index_groups_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_owner_uuid ON public.groups USING btree (owner_uuid);


--
-- Name: index_groups_on_owner_uuid_and_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_groups_on_owner_uuid_and_name ON public.groups USING btree (owner_uuid, name) WHERE (is_trashed = false);


--
-- Name: index_groups_on_trash_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_groups_on_trash_at ON public.groups USING btree (trash_at);


--
-- Name: index_groups_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_groups_on_uuid ON public.groups USING btree (uuid);


--
-- Name: index_humans_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_humans_on_owner_uuid ON public.humans USING btree (owner_uuid);


--
-- Name: index_humans_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_humans_on_uuid ON public.humans USING btree (uuid);


--
-- Name: index_job_tasks_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_created_at ON public.job_tasks USING btree (created_at);


--
-- Name: index_job_tasks_on_created_by_job_task_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_created_by_job_task_uuid ON public.job_tasks USING btree (created_by_job_task_uuid);


--
-- Name: index_job_tasks_on_job_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_job_uuid ON public.job_tasks USING btree (job_uuid);


--
-- Name: index_job_tasks_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_modified_at ON public.job_tasks USING btree (modified_at);


--
-- Name: index_job_tasks_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_owner_uuid ON public.job_tasks USING btree (owner_uuid);


--
-- Name: index_job_tasks_on_sequence; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_sequence ON public.job_tasks USING btree (sequence);


--
-- Name: index_job_tasks_on_success; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_job_tasks_on_success ON public.job_tasks USING btree (success);


--
-- Name: index_job_tasks_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_job_tasks_on_uuid ON public.job_tasks USING btree (uuid);


--
-- Name: index_jobs_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_created_at ON public.jobs USING btree (created_at);


--
-- Name: index_jobs_on_finished_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_finished_at ON public.jobs USING btree (finished_at);


--
-- Name: index_jobs_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_modified_at ON public.jobs USING btree (modified_at);


--
-- Name: index_jobs_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_modified_at_uuid ON public.jobs USING btree (modified_at DESC, uuid);


--
-- Name: index_jobs_on_output; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_output ON public.jobs USING btree (output);


--
-- Name: index_jobs_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_owner_uuid ON public.jobs USING btree (owner_uuid);


--
-- Name: index_jobs_on_script; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_script ON public.jobs USING btree (script);


--
-- Name: index_jobs_on_script_parameters_digest; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_script_parameters_digest ON public.jobs USING btree (script_parameters_digest);


--
-- Name: index_jobs_on_started_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_jobs_on_started_at ON public.jobs USING btree (started_at);


--
-- Name: index_jobs_on_submit_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_jobs_on_submit_id ON public.jobs USING btree (submit_id);


--
-- Name: index_jobs_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_jobs_on_uuid ON public.jobs USING btree (uuid);


--
-- Name: index_keep_disks_on_filesystem_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_keep_disks_on_filesystem_uuid ON public.keep_disks USING btree (filesystem_uuid);


--
-- Name: index_keep_disks_on_last_ping_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_keep_disks_on_last_ping_at ON public.keep_disks USING btree (last_ping_at);


--
-- Name: index_keep_disks_on_node_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_keep_disks_on_node_uuid ON public.keep_disks USING btree (node_uuid);


--
-- Name: index_keep_disks_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_keep_disks_on_owner_uuid ON public.keep_disks USING btree (owner_uuid);


--
-- Name: index_keep_disks_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_keep_disks_on_uuid ON public.keep_disks USING btree (uuid);


--
-- Name: index_keep_services_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_keep_services_on_owner_uuid ON public.keep_services USING btree (owner_uuid);


--
-- Name: index_keep_services_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_keep_services_on_uuid ON public.keep_services USING btree (uuid);


--
-- Name: index_links_on_created_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_created_at_and_uuid ON public.links USING btree (created_at, uuid);


--
-- Name: index_links_on_head_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_head_uuid ON public.links USING btree (head_uuid);


--
-- Name: index_links_on_modified_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_modified_at_and_uuid ON public.links USING btree (modified_at, uuid);


--
-- Name: index_links_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_owner_uuid ON public.links USING btree (owner_uuid);


--
-- Name: index_links_on_substring_head_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_substring_head_uuid ON public.links USING btree ("substring"((head_uuid)::text, 7, 5));


--
-- Name: index_links_on_substring_tail_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_substring_tail_uuid ON public.links USING btree ("substring"((tail_uuid)::text, 7, 5));


--
-- Name: index_links_on_tail_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_tail_uuid ON public.links USING btree (tail_uuid);


--
-- Name: index_links_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_links_on_uuid ON public.links USING btree (uuid);


--
-- Name: index_logs_on_created_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_created_at_and_uuid ON public.logs USING btree (created_at, uuid);


--
-- Name: index_logs_on_event_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_event_at ON public.logs USING btree (event_at);


--
-- Name: index_logs_on_event_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_event_type ON public.logs USING btree (event_type);


--
-- Name: index_logs_on_modified_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_modified_at_and_uuid ON public.logs USING btree (modified_at, uuid);


--
-- Name: index_logs_on_object_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_object_owner_uuid ON public.logs USING btree (object_owner_uuid);


--
-- Name: index_logs_on_object_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_object_uuid ON public.logs USING btree (object_uuid);


--
-- Name: index_logs_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_owner_uuid ON public.logs USING btree (owner_uuid);


--
-- Name: index_logs_on_summary; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_logs_on_summary ON public.logs USING btree (summary);


--
-- Name: index_logs_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_logs_on_uuid ON public.logs USING btree (uuid);


--
-- Name: index_nodes_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_nodes_on_created_at ON public.nodes USING btree (created_at);


--
-- Name: index_nodes_on_hostname; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_nodes_on_hostname ON public.nodes USING btree (hostname);


--
-- Name: index_nodes_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_nodes_on_modified_at ON public.nodes USING btree (modified_at);


--
-- Name: index_nodes_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_nodes_on_owner_uuid ON public.nodes USING btree (owner_uuid);


--
-- Name: index_nodes_on_slot_number; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_nodes_on_slot_number ON public.nodes USING btree (slot_number);


--
-- Name: index_nodes_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_nodes_on_uuid ON public.nodes USING btree (uuid);


--
-- Name: index_pipeline_instances_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_instances_on_created_at ON public.pipeline_instances USING btree (created_at);


--
-- Name: index_pipeline_instances_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_instances_on_modified_at ON public.pipeline_instances USING btree (modified_at);


--
-- Name: index_pipeline_instances_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_instances_on_modified_at_uuid ON public.pipeline_instances USING btree (modified_at DESC, uuid);


--
-- Name: index_pipeline_instances_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_instances_on_owner_uuid ON public.pipeline_instances USING btree (owner_uuid);


--
-- Name: index_pipeline_instances_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_pipeline_instances_on_uuid ON public.pipeline_instances USING btree (uuid);


--
-- Name: index_pipeline_templates_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_templates_on_created_at ON public.pipeline_templates USING btree (created_at);


--
-- Name: index_pipeline_templates_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_templates_on_modified_at ON public.pipeline_templates USING btree (modified_at);


--
-- Name: index_pipeline_templates_on_modified_at_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_templates_on_modified_at_uuid ON public.pipeline_templates USING btree (modified_at DESC, uuid);


--
-- Name: index_pipeline_templates_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pipeline_templates_on_owner_uuid ON public.pipeline_templates USING btree (owner_uuid);


--
-- Name: index_pipeline_templates_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_pipeline_templates_on_uuid ON public.pipeline_templates USING btree (uuid);


--
-- Name: index_repositories_on_created_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_repositories_on_created_at_and_uuid ON public.repositories USING btree (created_at, uuid);


--
-- Name: index_repositories_on_modified_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_repositories_on_modified_at_and_uuid ON public.repositories USING btree (modified_at, uuid);


--
-- Name: index_repositories_on_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_repositories_on_name ON public.repositories USING btree (name);


--
-- Name: index_repositories_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_repositories_on_owner_uuid ON public.repositories USING btree (owner_uuid);


--
-- Name: index_repositories_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_repositories_on_uuid ON public.repositories USING btree (uuid);


--
-- Name: index_specimens_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_specimens_on_created_at ON public.specimens USING btree (created_at);


--
-- Name: index_specimens_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_specimens_on_modified_at ON public.specimens USING btree (modified_at);


--
-- Name: index_specimens_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_specimens_on_owner_uuid ON public.specimens USING btree (owner_uuid);


--
-- Name: index_specimens_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_specimens_on_uuid ON public.specimens USING btree (uuid);


--
-- Name: index_traits_on_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_traits_on_name ON public.traits USING btree (name);


--
-- Name: index_traits_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_traits_on_owner_uuid ON public.traits USING btree (owner_uuid);


--
-- Name: index_traits_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_traits_on_uuid ON public.traits USING btree (uuid);


--
-- Name: index_trashed_groups_on_group_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_trashed_groups_on_group_uuid ON public.trashed_groups USING btree (group_uuid);


--
-- Name: index_users_on_created_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_created_at_and_uuid ON public.users USING btree (created_at, uuid);


--
-- Name: index_users_on_identity_url; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_identity_url ON public.users USING btree (identity_url);


--
-- Name: index_users_on_modified_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_modified_at_and_uuid ON public.users USING btree (modified_at, uuid);


--
-- Name: index_users_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_owner_uuid ON public.users USING btree (owner_uuid);


--
-- Name: index_users_on_username; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_username ON public.users USING btree (username);


--
-- Name: index_users_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_uuid ON public.users USING btree (uuid);


--
-- Name: index_virtual_machines_on_created_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_virtual_machines_on_created_at_and_uuid ON public.virtual_machines USING btree (created_at, uuid);


--
-- Name: index_virtual_machines_on_hostname; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_virtual_machines_on_hostname ON public.virtual_machines USING btree (hostname);


--
-- Name: index_virtual_machines_on_modified_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_virtual_machines_on_modified_at_and_uuid ON public.virtual_machines USING btree (modified_at, uuid);


--
-- Name: index_virtual_machines_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_virtual_machines_on_owner_uuid ON public.virtual_machines USING btree (owner_uuid);


--
-- Name: index_virtual_machines_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_virtual_machines_on_uuid ON public.virtual_machines USING btree (uuid);


--
-- Name: index_workflows_on_created_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_workflows_on_created_at_and_uuid ON public.workflows USING btree (created_at, uuid);


--
-- Name: index_workflows_on_modified_at_and_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_workflows_on_modified_at_and_uuid ON public.workflows USING btree (modified_at, uuid);


--
-- Name: index_workflows_on_owner_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_workflows_on_owner_uuid ON public.workflows USING btree (owner_uuid);


--
-- Name: index_workflows_on_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_workflows_on_uuid ON public.workflows USING btree (uuid);


--
-- Name: job_tasks_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX job_tasks_search_index ON public.job_tasks USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, job_uuid, created_by_job_task_uuid);


--
-- Name: jobs_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX jobs_search_index ON public.jobs USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, submit_id, script, script_version, cancelled_by_client_uuid, cancelled_by_user_uuid, output, is_locked_by_uuid, log, repository, supplied_script_version, docker_image_locator, state, arvados_sdk_version);


--
-- Name: jobs_trgm_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX jobs_trgm_text_search_idx ON public.jobs USING gin (((((((((((((((((((((((((((((((((((((((((((((COALESCE(uuid, ''::character varying))::text || ' '::text) || (COALESCE(owner_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(submit_id, ''::character varying))::text) || ' '::text) || (COALESCE(script, ''::character varying))::text) || ' '::text) || (COALESCE(script_version, ''::character varying))::text) || ' '::text) || COALESCE(script_parameters, ''::text)) || ' '::text) || (COALESCE(cancelled_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(cancelled_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(output, ''::character varying))::text) || ' '::text) || (COALESCE(is_locked_by_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(log, ''::character varying))::text) || ' '::text) || COALESCE(tasks_summary, ''::text)) || ' '::text) || COALESCE(runtime_constraints, ''::text)) || ' '::text) || (COALESCE(repository, ''::character varying))::text) || ' '::text) || (COALESCE(supplied_script_version, ''::character varying))::text) || ' '::text) || (COALESCE(docker_image_locator, ''::character varying))::text) || ' '::text) || (COALESCE(description, ''::character varying))::text) || ' '::text) || (COALESCE(state, ''::character varying))::text) || ' '::text) || (COALESCE(arvados_sdk_version, ''::character varying))::text) || ' '::text) || COALESCE(components, ''::text))) public.gin_trgm_ops);


--
-- Name: keep_disks_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX keep_disks_search_index ON public.keep_disks USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, ping_secret, node_uuid, filesystem_uuid, keep_service_uuid);


--
-- Name: keep_services_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX keep_services_search_index ON public.keep_services USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, service_host, service_type);


--
-- Name: links_index_on_properties; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX links_index_on_properties ON public.links USING gin (properties);


--
-- Name: links_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX links_search_index ON public.links USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, tail_uuid, link_class, name, head_uuid);


--
-- Name: links_tail_name_unique_if_link_class_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX links_tail_name_unique_if_link_class_name ON public.links USING btree (tail_uuid, name) WHERE ((link_class)::text = 'name'::text);


--
-- Name: logs_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX logs_search_index ON public.logs USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, object_uuid, event_type, object_owner_uuid);


--
-- Name: nodes_index_on_info; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX nodes_index_on_info ON public.nodes USING gin (info);


--
-- Name: nodes_index_on_properties; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX nodes_index_on_properties ON public.nodes USING gin (properties);


--
-- Name: nodes_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX nodes_search_index ON public.nodes USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, hostname, domain, ip_address, job_uuid);


--
-- Name: permission_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX permission_target ON public.materialized_permissions USING btree (target_uuid);


--
-- Name: permission_user_target; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX permission_user_target ON public.materialized_permissions USING btree (user_uuid, target_uuid);


--
-- Name: pipeline_instances_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pipeline_instances_search_index ON public.pipeline_instances USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, pipeline_template_uuid, name, state);


--
-- Name: pipeline_instances_trgm_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pipeline_instances_trgm_text_search_idx ON public.pipeline_instances USING gin (((((((((((((((((((((((COALESCE(uuid, ''::character varying))::text || ' '::text) || (COALESCE(owner_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(pipeline_template_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(name, ''::character varying))::text) || ' '::text) || COALESCE(components, ''::text)) || ' '::text) || COALESCE(properties, ''::text)) || ' '::text) || (COALESCE(state, ''::character varying))::text) || ' '::text) || COALESCE(components_summary, ''::text)) || ' '::text) || (COALESCE(description, ''::character varying))::text)) public.gin_trgm_ops);


--
-- Name: pipeline_template_owner_uuid_name_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX pipeline_template_owner_uuid_name_unique ON public.pipeline_templates USING btree (owner_uuid, name);


--
-- Name: pipeline_templates_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pipeline_templates_search_index ON public.pipeline_templates USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name);


--
-- Name: pipeline_templates_trgm_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pipeline_templates_trgm_text_search_idx ON public.pipeline_templates USING gin (((((((((((((((COALESCE(uuid, ''::character varying))::text || ' '::text) || (COALESCE(owner_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(name, ''::character varying))::text) || ' '::text) || COALESCE(components, ''::text)) || ' '::text) || (COALESCE(description, ''::character varying))::text)) public.gin_trgm_ops);


--
-- Name: repositories_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repositories_search_index ON public.repositories USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name);


--
-- Name: specimens_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX specimens_search_index ON public.specimens USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, material);


--
-- Name: traits_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX traits_search_index ON public.traits USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name);


--
-- Name: unique_schema_migrations; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX unique_schema_migrations ON public.schema_migrations USING btree (version);


--
-- Name: users_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX users_search_index ON public.users USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, email, first_name, last_name, identity_url, default_owner_uuid, username, redirect_to_user_uuid);


--
-- Name: virtual_machines_search_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX virtual_machines_search_index ON public.virtual_machines USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, hostname);


--
-- Name: workflows_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workflows_search_idx ON public.workflows USING btree (uuid, owner_uuid, modified_by_client_uuid, modified_by_user_uuid, name);


--
-- Name: workflows_trgm_text_search_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workflows_trgm_text_search_idx ON public.workflows USING gin (((((((((((((COALESCE(uuid, ''::character varying))::text || ' '::text) || (COALESCE(owner_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_client_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(modified_by_user_uuid, ''::character varying))::text) || ' '::text) || (COALESCE(name, ''::character varying))::text) || ' '::text) || COALESCE(description, ''::text))) public.gin_trgm_ops);


--
-- PostgreSQL database dump complete
--

SET search_path TO "$user", public;

INSERT INTO "schema_migrations" (version) VALUES
('20121016005009'),
('20130105203021'),
('20130105224358'),
('20130105224618'),
('20130107181109'),
('20130107212832'),
('20130109175700'),
('20130109220548'),
('20130113214204'),
('20130116024233'),
('20130116215213'),
('20130118002239'),
('20130122020042'),
('20130122201442'),
('20130122221616'),
('20130123174514'),
('20130123180224'),
('20130123180228'),
('20130125220425'),
('20130128202518'),
('20130128231343'),
('20130130205749'),
('20130203104818'),
('20130203104824'),
('20130203115329'),
('20130207195855'),
('20130218181504'),
('20130226170000'),
('20130313175417'),
('20130315155820'),
('20130315183626'),
('20130315213205'),
('20130318002138'),
('20130319165853'),
('20130319180730'),
('20130319194637'),
('20130319201431'),
('20130319235957'),
('20130320000107'),
('20130326173804'),
('20130326182917'),
('20130415020241'),
('20130425024459'),
('20130425214427'),
('20130523060112'),
('20130523060213'),
('20130524042319'),
('20130528134100'),
('20130606183519'),
('20130608053730'),
('20130610202538'),
('20130611163736'),
('20130612042554'),
('20130617150007'),
('20130626002829'),
('20130626022810'),
('20130627154537'),
('20130627184333'),
('20130708163414'),
('20130708182912'),
('20130708185153'),
('20130724153034'),
('20131007180607'),
('20140117231056'),
('20140124222114'),
('20140129184311'),
('20140317135600'),
('20140319160547'),
('20140321191343'),
('20140324024606'),
('20140325175653'),
('20140402001908'),
('20140407184311'),
('20140421140924'),
('20140421151939'),
('20140421151940'),
('20140422011506'),
('20140423132913'),
('20140423133559'),
('20140501165548'),
('20140519205916'),
('20140527152921'),
('20140530200539'),
('20140601022548'),
('20140602143352'),
('20140607150616'),
('20140611173003'),
('20140627210837'),
('20140709172343'),
('20140714184006'),
('20140811184643'),
('20140817035914'),
('20140818125735'),
('20140826180337'),
('20140828141043'),
('20140909183946'),
('20140911221252'),
('20140918141529'),
('20140918153541'),
('20140918153705'),
('20140924091559'),
('20141111133038'),
('20141208164553'),
('20141208174553'),
('20141208174653'),
('20141208185217'),
('20150122175935'),
('20150123142953'),
('20150203180223'),
('20150206210804'),
('20150206230342'),
('20150216193428'),
('20150303210106'),
('20150312151136'),
('20150317132720'),
('20150324152204'),
('20150423145759'),
('20150512193020'),
('20150526180251'),
('20151202151426'),
('20151215134304'),
('20151229214707'),
('20160208210629'),
('20160209155729'),
('20160324144017'),
('20160506175108'),
('20160509143250'),
('20160808151559'),
('20160819195557'),
('20160819195725'),
('20160901210110'),
('20160909181442'),
('20160926194129'),
('20161019171346'),
('20161111143147'),
('20161115171221'),
('20161115174218'),
('20161213172944'),
('20161222153434'),
('20161223090712'),
('20170102153111'),
('20170105160301'),
('20170105160302'),
('20170216170823'),
('20170301225558'),
('20170319063406'),
('20170328215436'),
('20170330012505'),
('20170419173031'),
('20170419173712'),
('20170419175801'),
('20170628185847'),
('20170704160233'),
('20170706141334'),
('20170824202826'),
('20170906224040'),
('20171027183824'),
('20171208203841'),
('20171212153352'),
('20180216203422'),
('20180228220311'),
('20180313180114'),
('20180501182859'),
('20180514135529'),
('20180607175050'),
('20180608123145'),
('20180806133039'),
('20180820130357'),
('20180820132617'),
('20180820135808'),
('20180824152014'),
('20180824155207'),
('20180904110712'),
('20180913175443'),
('20180915155335'),
('20180917200000'),
('20180917205609'),
('20180919001158'),
('20181001175023'),
('20181004131141'),
('20181005192222'),
('20181011184200'),
('20181213183234'),
('20190214214814'),
('20190322174136'),
('20190422144631'),
('20190523180148'),
('20190808145904'),
('20190809135453'),
('20190905151603'),
('20200501150153'),
('20200602141328'),
('20200914203202'),
('20201103170213'),
('20201105190435'),
('20201202174753'),
('20210108033940'),
('20210126183521'),
('20210621204455'),
('20210816191509'),
('20211027154300'),
('20220224203102'),
('20220301155729'),
('20220303204419'),
('20220401153101'),
('20220505112900');


