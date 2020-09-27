
include!(concat!(env!("OUT_DIR"), "/arvados-api.rs"));




use reqwest::{Request, Url, Client};
use reqwest::header::{HeaderValue, HeaderMap, AUTHORIZATION};

pub struct AravadosClient {
    http_client: Client,
    base_url: String,
}

type ArvadosError = Box<dyn std::error::Error + Send + Sync>;
pub type Result<T> = std::result::Result<T, ArvadosError>;


impl<'a> AravadosClient {
    pub fn new(arv_api_host: &str, arv_api_token: &str, arv_api_host_insecure: bool) -> Result<Self> {
        let proto = if arv_api_host_insecure { "http" } else {"https" };
        let base_url = format!("{}://{}/arvados/v1/", proto, arv_api_host);
        let mut headers = HeaderMap::new();
        let auth = format!("OAuth2 {}", arv_api_token);
        headers.insert(AUTHORIZATION, HeaderValue::from_str(auth.as_ref())?);
        let http_client = Client::builder().default_headers(headers).build()?;
        Ok(Self { http_client, base_url })
    }

    pub fn keep_services(&self) -> ArvadosKeepServices {
        ArvadosKeepServices { client: self }
    }

    pub fn http_client(&self) -> &Client {
        &self.http_client
    }
}

pub struct ArvadosKeepServices<'a> {
    client: &'a AravadosClient,
}

impl<'a> ArvadosKeepServices<'a> {
    // uuid is a required parameter.
    pub fn get<T : AsRef<str>>(&self, uuid: T) -> ArvadosKeepServicesGet {
        ArvadosKeepServicesGet { client: self.client, uuid: uuid.as_ref().to_string() }
    }
}

pub struct ArvadosKeepServicesGet<'a> {
    client: &'a AravadosClient,
    uuid: String,
}

impl<'a> ArvadosKeepServicesGet<'a> {
    /// Return a request object for this service.
    pub fn request(&self) -> Result<Request> {
        let url : Url = Url::parse(format!("{}keep_services/{}", self.client.base_url, self.uuid).as_ref())?;
        let req = self.client.http_client.get(url).build()?;
        Ok(req)
    }
}


#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn it_works() -> Result<()> {
        let arv_api_host = "localhost";
        let arv_api_token = "token";
        let arv_api_host_insecure = false;

        let arvados = AravadosClient::new(arv_api_host, arv_api_token, arv_api_host_insecure)?;
        let req = arvados.keep_services().get("xyz").request()?;

        assert_eq!(format!("{:?}", req), "Request { method: GET, url: \"https://localhost/arvados/v1/keep_services/xyz\", headers: {} }");
        Ok(())
    }
}
