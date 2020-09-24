//!
//!  This file was edited down from the generated code at
//!  https://github.com/Byron/google-apis-rs/blob/master/gen/discovery1/src/lib.rs
//!
//! This that are wrong with this:
//! * The data structures are not separated from the code.
//! * The code is not async.
//! * The code is single threaded.
//!
//! Given more time we might be able to generate the structures from a mako
//! script, but for now we are stealing.
//!

//#![allow(dead_code)]

use serde::Deserialize;
use std::collections::HashMap;

/// OAuth 2.0 authentication information.
/// 
/// 
#[derive(Deserialize, Debug)]
pub struct RestDescriptionAuthOauth2 {
    /// Available OAuth 2.0 scopes.
    pub scopes: Option<HashMap<String, RestDescriptionAuthOauth2Scopes>>,
}

/// The schema for the response.
/// 

/// 
#[derive(Deserialize, Debug)]
pub struct RestMethodResponse {
    /// Schema ID for the response schema.
    #[serde(rename="$ref")]
    pub ref_: Option<String>,
}

/// In a variant data type, the value of one property is used to determine how to interpret the entire entity. Its value must exist in a map of descriminant values to schema names.
/// 
#[derive(Deserialize, Debug)]
pub struct JsonSchemaVariant {
    /// The map of discriminant value to schema to use for parsing..
    pub map: Option<Vec<JsonSchemaVariantMap>>,
    /// The name of the type discriminant property.
    pub discriminant: Option<String>,
}

/// Supported upload protocols.
/// 
#[derive(Deserialize, Debug)]
pub struct RestMethodMediaUploadProtocols {
    /// Supports uploading as a single HTTP request.
    pub simple: Option<RestMethodMediaUploadProtocolsSimple>,
    /// Supports the Resumable Media Upload protocol.
    pub resumable: Option<RestMethodMediaUploadProtocolsResumable>,
}

/// Supports the Resumable Media Upload protocol.
/// 

/// 
#[derive(Deserialize, Debug)]
pub struct RestMethodMediaUploadProtocolsResumable {
    /// The URI path to be used for upload. Should be used in conjunction with the basePath property at the api-level.
    pub path: Option<String>,
    /// True if this endpoint supports uploading multipart media.
    pub multipart: Option<bool>,
}

/// Additional information about this property.
/// 

/// 
#[derive(Deserialize, Debug)]
pub struct JsonSchemaAnnotations {
    /// A list of methods for which this property is required on requests.
    pub required: Option<Vec<String>>,
}

/// The map of discriminant value to schema to use for parsing..
/// 

/// 
#[derive(Deserialize, Debug)]
pub struct JsonSchemaVariantMap {
    /// no description provided
    pub type_value: Option<String>,
    /// no description provided
    #[serde(rename="$ref")]
    pub ref_: Option<String>,
}

/// Links to 16x16 and 32x32 icons representing the API.
/// 

/// 
#[derive(Deserialize, Debug)]
pub struct RestDescriptionIcons {
    /// The URL of the 32x32 icon.
    pub x32: Option<String>,
    /// The URL of the 16x16 icon.
    pub x16: Option<String>,
}


/// 

/// 
#[derive(Deserialize, Debug)]
pub struct RestMethod {
    /// OAuth 2.0 scopes applicable to this method.
    pub scopes: Option<Vec<String>>,
    /// Description of this method.
    pub description: Option<String>,
    /// Details for all parameters in this method.
    pub parameters: Option<HashMap<String, JsonSchema>>,
    /// Whether this method supports media uploads.
    #[serde(rename="supportsMediaUpload")]
    pub supports_media_upload: Option<bool>,
    /// Whether this method requires an ETag to be specified. The ETag is sent as an HTTP If-Match or If-None-Match header.
    #[serde(rename="etagRequired")]
    pub etag_required: Option<bool>,
    /// Media upload parameters.
    #[serde(rename="mediaUpload")]
    pub media_upload: Option<RestMethodMediaUpload>,
    /// The schema for the request.
    pub request: Option<RestMethodRequest>,
    /// Indicates that downloads from this method should use the download service URL (i.e. "/download"). Only applies if the method supports media download.
    #[serde(rename="useMediaDownloadService")]
    pub use_media_download_service: Option<bool>,
    /// HTTP method used by this method.
    #[serde(rename="httpMethod")]
    pub http_method: Option<String>,
    /// Whether this method supports subscriptions.
    #[serde(rename="supportsSubscription")]
    pub supports_subscription: Option<bool>,
    /// Ordered list of required parameters, serves as a hint to clients on how to structure their method signatures. The array is ordered such that the "most-significant" parameter appears first.
    #[serde(rename="parameterOrder")]
    pub parameter_order: Option<Vec<String>>,
    /// A unique ID for this method. This property can be used to match methods between different versions of Discovery.
    pub id: Option<String>,
    /// The URI path of this REST method. Should be used in conjunction with the basePath property at the api-level.
    pub path: Option<String>,
    /// The schema for the response.
    pub response: Option<RestMethodResponse>,
    /// Whether this method supports media downloads.
    #[serde(rename="supportsMediaDownload")]
    pub supports_media_download: Option<bool>,
}


/// 
/// # Activities
/// 
/// This type is used in activities, which are methods you may call on this type or where this type is involved in. 
/// The list links the activity name, along with information about where it is used (one of *request* and *response*).
/// 
/// * [get rest apis](struct.ApiGetRestCall.html) (response)
/// 
#[derive(Deserialize, Debug)]
pub struct RestDescription {
    /// The protocol described by this document.
    pub protocol: Option<String>,
    /// API-level methods for this API.
    pub methods: Option<HashMap<String, RestMethod>>,
    /// Labels for the status of this API, such as labs or deprecated.
    pub labels: Option<Vec<String>>,
    /// The kind for this response.
    pub kind: Option<String>,
    /// Indicates how the API name should be capitalized and split into various parts. Useful for generating pretty class names.
    #[serde(rename="canonicalName")]
    pub canonical_name: Option<String>,
    /// A link to human readable documentation for the API.
    #[serde(rename="documentationLink")]
    pub documentation_link: Option<String>,
    /// The name of the owner of this API. See ownerDomain.
    #[serde(rename="ownerName")]
    pub owner_name: Option<String>,
    /// The package of the owner of this API. See ownerDomain.
    #[serde(rename="packagePath")]
    pub package_path: Option<String>,
    /// The path for REST batch requests.
    #[serde(rename="batchPath")]
    pub batch_path: Option<String>,
    /// The ID of this API.
    pub id: Option<String>,
    /// A list of supported features for this API.
    pub features: Option<Vec<String>>,
    /// The domain of the owner of this API. Together with the ownerName and a packagePath values, this can be used to generate a library for this API which would have a unique fully qualified name.
    #[serde(rename="ownerDomain")]
    pub owner_domain: Option<String>,
    /// The root URL under which all API services live.
    #[serde(rename="rootUrl")]
    pub root_url: Option<String>,
    /// The name of this API.
    pub name: Option<String>,
    /// Common parameters that apply across all apis.
    pub parameters: Option<HashMap<String, JsonSchema>>,
    /// Links to 16x16 and 32x32 icons representing the API.
    pub icons: Option<RestDescriptionIcons>,
    /// no description provided
    pub version_module: Option<bool>,
    /// The description of this API.
    pub description: Option<String>,
    /// The title of this API.
    pub title: Option<String>,
    /// Enable exponential backoff for suitable methods in the generated clients.
    #[serde(rename="exponentialBackoffDefault")]
    pub exponential_backoff_default: Option<bool>,
    /// [DEPRECATED] The base URL for REST requests.
    #[serde(rename="baseUrl")]
    pub base_url: Option<String>,
    /// The ETag for this response.
    pub etag: Option<String>,
    /// The version of this API.
    pub version: Option<String>,
    /// The base path for all REST requests.
    #[serde(rename="servicePath")]
    pub service_path: Option<String>,
    /// Indicate the version of the Discovery API used to generate this doc.
    #[serde(rename="discoveryVersion")]
    pub discovery_version: Option<String>,
    /// The schemas for this API.
    pub schemas: Option<HashMap<String, JsonSchema>>,
    /// Authentication information.
    pub auth: Option<RestDescriptionAuth>,
    /// [DEPRECATED] The base path for REST requests.
    #[serde(rename="basePath")]
    pub base_path: Option<String>,
    /// The resources in this API.
    pub resources: Option<HashMap<String, RestResource>>,
    /// The version of this API.
    pub revision: Option<String>,
}

/// Media upload parameters.
/// 

/// 
#[derive(Deserialize, Debug)]
pub struct RestMethodMediaUpload {
    /// Maximum size of a media upload, such as "1MB", "2GB" or "3TB".
    #[serde(rename="maxSize")]
    pub max_size: Option<String>,
    /// MIME Media Ranges for acceptable media uploads to this method.
    pub accept: Option<Vec<String>>,
    /// Supported upload protocols.
    pub protocols: Option<RestMethodMediaUploadProtocols>,
}


/// 
/// # Activities
/// 
/// This type is used in activities, which are methods you may call on this type or where this type is involved in. 
/// The list links the activity name, along with information about where it is used (one of *request* and *response*).
/// 
/// * [list apis](struct.ApiListCall.html) (response)
/// 
#[derive(Deserialize, Debug)]
pub struct DirectoryList {
    /// The individual directory entries. One entry per api/version pair.
    pub items: Option<Vec<DirectoryListItems>>,
    /// Indicate the version of the Discovery API used to generate this doc.
    #[serde(rename="discoveryVersion")]
    pub discovery_version: Option<String>,
    /// The kind for this response.
    pub kind: Option<String>,
}

/// 
/// 
#[derive(Deserialize, Debug)]
pub struct JsonSchema {
    /// A description of this object.
    pub description: Option<String>,
    /// An additional regular expression or key that helps constrain the value. For more details see: http://tools.ietf.org/html/draft-zyp-json-schema-03#section-5.23
    pub format: Option<String>,
    /// Values this parameter may take (if it is an enum).
    #[serde(rename="enum")]
    pub enum_: Option<Vec<String>>,
    /// In a variant data type, the value of one property is used to determine how to interpret the entire entity. Its value must exist in a map of descriminant values to schema names.
    pub variant: Option<JsonSchemaVariant>,
    /// The descriptions for the enums. Each position maps to the corresponding value in the "enum" array.
    #[serde(rename="enumDescriptions")]
    pub enum_descriptions: Option<Vec<String>>,
    /// The value is read-only, generated by the service. The value cannot be modified by the client. If the value is included in a POST, PUT, or PATCH request, it is ignored by the service.
    #[serde(rename="readOnly")]
    pub read_only: Option<bool>,
    /// The minimum value of this parameter.
    pub minimum: Option<String>,
    /// Whether this parameter may appear multiple times.
    pub repeated: Option<bool>,
    /// Unique identifier for this schema.
    pub id: Option<String>,
    /// A reference to another schema. The value of this property is the "id" of another schema.
    #[serde(rename="$ref")]
    pub ref_: Option<String>,
    /// The default value of this property (if one exists).
    pub default: Option<String>,
    /// If this is a schema for an array, this property is the schema for each element in the array.
    pub items: Option<Option<Box<JsonSchema>>>,
    /// Whether the parameter is required.
    pub required: Option<bool>,
    /// The maximum value of this parameter.
    pub maximum: Option<String>,
    /// If this is a schema for an object, list the schema for each property of this object.
    pub properties: Option<HashMap<String, JsonSchema>>,
    /// Whether this parameter goes in the query or the path for REST requests.
    pub location: Option<String>,
    /// The regular expression this parameter must conform to. Uses Java 6 regex format: http://docs.oracle.com/javase/6/docs/api/java/util/regex/Pattern.html
    pub pattern: Option<String>,
    /// If this is a schema for an object, this property is the schema for any additional properties with dynamic keys on this object.
    #[serde(rename="additionalProperties")]
    pub additional_properties: Option<Option<Box<JsonSchema>>>,
    /// The value type for this schema. A list of values can be found here: http://tools.ietf.org/html/draft-zyp-json-schema-03#section-5.1
    #[serde(rename="type")]
    pub type_: Option<String>,
    /// Additional information about this property.
    pub annotations: Option<JsonSchemaAnnotations>,
}


/// The individual directory entries. One entry per api/version pair.
/// 
/// 
#[derive(Deserialize, Debug)]
pub struct DirectoryListItems {
    /// The kind for this response.
    pub kind: Option<String>,
    /// The URL for the discovery REST document.
    #[serde(rename="discoveryRestUrl")]
    pub discovery_rest_url: Option<String>,
    /// The description of this API.
    pub description: Option<String>,
    /// Links to 16x16 and 32x32 icons representing the API.
    pub icons: Option<DirectoryListItemsIcons>,
    /// Labels for the status of this API, such as labs or deprecated.
    pub labels: Option<Vec<String>>,
    /// True if this version is the preferred version to use.
    pub preferred: Option<bool>,
    /// A link to the discovery document.
    #[serde(rename="discoveryLink")]
    pub discovery_link: Option<String>,
    /// The version of the API.
    pub version: Option<String>,
    /// The title of this API.
    pub title: Option<String>,
    /// A link to human readable documentation for the API.
    #[serde(rename="documentationLink")]
    pub documentation_link: Option<String>,
    /// The id of this API.
    pub id: Option<String>,
    /// The name of the API.
    pub name: Option<String>,
}

/// Authentication information.
/// 
/// 
#[derive(Deserialize, Debug)]
pub struct RestDescriptionAuth {
    /// OAuth 2.0 authentication information.
    pub oauth2: Option<RestDescriptionAuthOauth2>,
}

/// The scope value.
/// 
/// 
#[derive(Deserialize, Debug)]
pub struct RestDescriptionAuthOauth2Scopes {
    /// Description of scope.
    pub description: Option<String>,
}

/// Links to 16x16 and 32x32 icons representing the API.
/// 
/// 
#[derive(Deserialize, Debug)]
pub struct DirectoryListItemsIcons {
    /// The URL of the 32x32 icon.
    pub x32: Option<String>,
    /// The URL of the 16x16 icon.
    pub x16: Option<String>,
}





/// The schema for the request.
/// 
/// 
#[derive(Deserialize, Debug)]
pub struct RestMethodRequest {
    /// parameter name.
    #[serde(rename="parameterName")]
    pub parameter_name: Option<String>,
    /// Schema ID for the request schema.
    #[serde(rename="$ref")]
    pub ref_: Option<String>,
}

/// 
/// 
#[derive(Deserialize, Debug)]
pub struct RestResource {
    /// Methods on this resource.
    pub methods: Option<HashMap<String, RestMethod>>,
    /// Sub-resources on this resource.
    pub resources: Option<HashMap<String, RestResource>>,
}


/// Supports uploading as a single HTTP request.
/// 

/// 
#[derive(Deserialize, Debug)]
pub struct RestMethodMediaUploadProtocolsSimple {
    /// The URI path to be used for upload. Should be used in conjunction with the basePath property at the api-level.
    pub path: Option<String>,
    /// True if this endpoint supports upload multipart media.
    pub multipart: Option<bool>,
}


#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn testd1() -> Result<(), Box<dyn std::error::Error>> {
        let arvados_api_json = std::fs::File::open("arvados-api.json")?;
        let desc : RestDescription = serde_json::from_reader(arvados_api_json)?;
        //let desc_json = serde_json::to_vec_pretty(&desc);
        let mut t = std::fs::File::create("/tmp/1")?;
        use std::io::Write;
        write!(t, "{:#?}", desc)?;
        Ok(())
    }
}
