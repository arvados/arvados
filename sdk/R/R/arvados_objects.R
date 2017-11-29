#' Collection Object
#' 
#' @details 
#' Todo: Update description
#' Collection
#' 
#' @param uuid Object ID
#' @param etag Object version
#' @param owner_uuid No description
#' @param created_at No description
#' @param modified_by_client_uuid No description
#' @param modified_by_user_uuid No description
#' @param modified_at No description
#' @param portable_data_hash No description
#' @param replication_desired No description
#' @param replication_confirmed_at No description
#' @param replication_confirmed No description
#' @param updated_at No description
#' @param manifest_text No description
#' @param name No description
#' @param description No description
#' @param properties No description
#' @param delete_at No description
#' @param file_names No description
#' @param trash_at No description
#' @param is_trashed No description
#' 
#' @return Collection object
#' 
#' @family Collection functions
#' @export
Collection <- function(uuid                     = NULL,
                       etag                     = NULL,
                       owner_uuid               = NULL,
                       created_at               = NULL,
                       modified_by_client_uuid  = NULL,
                       modified_by_user_uuid    = NULL,
                       modified_at              = NULL,
                       portable_data_hash       = NULL,
                       replication_desired      = NULL,
                       replication_confirmed_at = NULL,
                       replication_confirmed    = NULL,
                       updated_at               = NULL,
                       manifest_text            = NULL,
                       name                     = NULL,
                       description              = NULL,
                       properties               = NULL,
                       delete_at                = NULL,
                       file_names               = NULL,
                       trash_at                 = NULL,
                       is_trashed               = NULL)
{
    structure(list(uuid                     = uuid,
                   etag                     = etag,
                   owner_uuid               = owner_uuid,
                   created_at               = created_at,
                   modified_by_client_uuid  = modified_by_client_uuid,
                   modified_by_user_uuid    = modified_by_user_uuid,
                   modified_at              = modified_at,
                   portable_data_hash       = portable_data_hash,
                   replication_desired      = replication_desired,
                   replication_confirmed_at = replication_confirmed_at,
                   replication_confirmed    = replication_confirmed,
                   updated_at               = updated_at,
                   manifest_text            = manifest_text,
                   name                     = name,
                   description              = description,
                   properties               = properties,
                   delete_at                = delete_at,
                   file_names               = file_names,
                   trash_at                 = trash_at,
                   is_trashed               = is_trashed),
              class = "ArvadosCollection")
}

