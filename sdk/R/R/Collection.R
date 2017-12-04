source("./R/Arvados.R")
source("./R/HttpParser.R")

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
#' @export
Collection <- setRefClass(

    "Collection",

    #NOTE(Fudo): Fix types!
    fields = list(uuid                     = "ANY",
                  items                    = "ANY",
                  etag                     = "ANY",
                  owner_uuid               = "ANY",
                  created_at               = "ANY",
                  modified_by_client_uuid  = "ANY",
                  modified_by_user_uuid    = "ANY",
                  modified_at              = "ANY",
                  portable_data_hash       = "ANY",
                  replication_desired      = "ANY",
                  replication_confirmed_at = "ANY",
                  replication_confirmed    = "ANY",
                  updated_at               = "ANY",
                  manifest_text            = "ANY",
                  name                     = "ANY",
                  description              = "ANY",
                  properties               = "ANY",
                  delete_at                = "ANY",
                  file_names               = "ANY",
                  trash_at                 = "ANY",
                  is_trashed               = "ANY",
                  arvados_api              = "Arvados"
    ),

    methods = list(

        initialize = function(api, uuid) 
        {
            arvados_api <<- api
            result <- arvados_api$collection_get(uuid)
            
            uuid                     <<- result$uuid                               
            etag                     <<- result$etag                               
            owner_uuid               <<- result$owner_uuid                         
            created_at               <<- result$created_at                         
            modified_by_client_uuid  <<- result$modified_by_client_uuid            
            modified_by_user_uuid    <<- result$modified_by_user_uuid              
            modified_at              <<- result$modified_at                        
            portable_data_hash       <<- result$portable_data_hash                 
            replication_desired      <<- result$replication_desired                
            replication_confirmed_at <<- result$replication_confirmed_at           
            replication_confirmed    <<- result$replication_confirmed              
            updated_at               <<- result$updated_at                         
            manifest_text            <<- result$manifest_text                      
            name                     <<- result$name                               
            description              <<- result$description                        
            properties               <<- result$properties                         
            delete_at                <<- result$delete_at                          
            file_names               <<- result$file_names                         
            trash_at                 <<- result$trash_at                           
            is_trashed               <<- result$is_trashed                         

            items  <<- getCollectionContent()
        },

        getCollectionContent = function()
        {
            #IMPORTANT(Fudo): This url is hardcoded for now. Fix it later.
            uri <- URLencode("https://collections.4xphq.arvadosapi.com/c=4xphq-4zz18-9d5b0qm4fgijeyi/_/")

            # fetch directory listing via curl and parse XML response
            h <- curl::new_handle()
            curl::handle_setopt(h, customrequest = "PROPFIND")

            #IMPORTANT(Fudo): Token is hardcoded as well. Write it properly.
            curl::handle_setheaders(h, "Authorization" = paste("OAuth2 4invqy35tf70t7hmvdc83ges8ug9cklhgqq1l8gj2cjn18teuq"))
            response <- curl::curl_fetch_memory(uri, h)

            HttpParser()$parseWebDAVResponse(response, uri)
        }
    )
)
