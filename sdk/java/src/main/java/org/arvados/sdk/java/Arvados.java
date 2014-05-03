package org.arvados.sdk.java;

import com.google.api.client.http.javanet.*;
import com.google.api.client.http.ByteArrayContent;
import com.google.api.client.http.GenericUrl;
import com.google.api.client.http.HttpContent;
import com.google.api.client.http.HttpRequest;
import com.google.api.client.http.HttpRequestFactory;
import com.google.api.client.http.HttpTransport;
import com.google.api.client.http.UriTemplate;
import com.google.api.client.json.JsonFactory;
import com.google.api.client.json.jackson2.JacksonFactory;
import com.google.api.client.util.Lists;
import com.google.api.client.util.Maps;
import com.google.api.services.discovery.Discovery;
import com.google.api.services.discovery.model.JsonSchema;
import com.google.api.services.discovery.model.RestDescription;
import com.google.api.services.discovery.model.RestMethod;
import com.google.api.services.discovery.model.RestResource;

import java.math.BigDecimal;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import org.apache.log4j.Logger;

/**
 * This class provides a java SDK interface to Arvados API server.
 * 
 * Please refer to http://doc.arvados.org/api/ to learn about the
 *  various resources and methods exposed by the API server.
 *  
 * @author radhika
 */
public class Arvados {
  // HttpTransport and JsonFactory are thread-safe. So, use global instances.
  private HttpTransport httpTransport;
  private final JsonFactory jsonFactory = JacksonFactory.getDefaultInstance();

  private String arvadosApiToken;
  private String arvadosApiHost;
  private boolean arvadosApiHostInsecure;

  private String arvadosRootUrl;

  private static final Logger logger = Logger.getLogger(Arvados.class);

  // Get it on a discover call and reuse on the call requests
  RestDescription restDescription = null;
  String apiName = null;
  String apiVersion = null;

  public Arvados (String apiName, String apiVersion){
    this (apiName, apiVersion, null, null, null);
  }

  public Arvados (String apiName, String apiVersion, String token, String host, String hostInsecure){
    try {
      this.apiName = apiName;
      this.apiVersion = apiVersion;

      // Read needed environmental variables if they are not passed
      if (token != null) {
        arvadosApiToken = token;
      } else {
        arvadosApiToken = System.getenv().get("ARVADOS_API_TOKEN");
        if (arvadosApiToken == null) {
          throw new Exception("Missing environment variable: ARVADOS_API_TOKEN");
        }
      }

      if (host != null) {
        arvadosApiHost = host;
      } else {
        arvadosApiHost = System.getenv().get("ARVADOS_API_HOST");      
        if (arvadosApiHost == null) {
          throw new Exception("Missing environment variable: ARVADOS_API_HOST");
        }
      }
      arvadosRootUrl = "https://" + arvadosApiHost;
      arvadosRootUrl += (arvadosApiHost.endsWith("/")) ? "" : "/";

      if (hostInsecure != null) {
        arvadosApiHostInsecure = Boolean.valueOf(hostInsecure);
      } else {
        arvadosApiHostInsecure = "true".equals(System.getenv().get("ARVADOS_API_HOST_INSECURE")) ? true : false;
      }

      // Create HTTP_TRANSPORT object
      NetHttpTransport.Builder builder = new NetHttpTransport.Builder();
      if (arvadosApiHostInsecure) {
        builder.doNotValidateCertificate();
      }
      httpTransport = builder.build();
    } catch (Throwable t) {
      t.printStackTrace();
    }
  }

  /**
   * Make a discover call and cache the response in-memory.
   *  Reload the document on each invocation.
   * @return RestDescription
   * @throws Exception
   */
  public RestDescription discover() throws Exception {
    restDescription = loadArvadosApi();

    // compute method details
    ArrayList<MethodDetails> result = Lists.newArrayList();
    String resourceName = "";
    processResources(result, resourceName, restDescription.getResources());

    // display method details
    Collections.sort(result);
    StringBuffer buffer = new StringBuffer();
    for (MethodDetails methodDetail : result) {
      buffer.append("\nArvados call " + apiName + " " + apiVersion + " " + methodDetail.name);
      for (String param : methodDetail.requiredParameters) {
        buffer.append(" <" + param + ">");
      }
      if (methodDetail.hasContent) {
        buffer.append(" contentFile");
      }
      if (methodDetail.optionalParameters.isEmpty() && !methodDetail.hasContent) {
        buffer.append("\n");
      } else {
        buffer.append("\n [optional parameters...]");
        buffer.append("\n  --contentType <value> (default is \"application/json\")");
        for (String param : methodDetail.optionalParameters) {
          buffer.append("\n  --" + param + " <value>");
        }
      }
    }
    logger.debug(buffer.toString());

    return (restDescription);
  }

  /**
   * Make a call to API server with the provide call information.
   * @param resourceName
   * @param methodName
   * @param paramsMap
   * @return Object
   * @throws Exception
   */
  public String call(String resourceName, String methodName, Map<String, Object> paramsMap) throws Exception {
    RestMethod method = getMatchingMethod(resourceName, methodName);

    HashMap<String, Object> parameters = loadParameters(paramsMap, method);

    GenericUrl url = new GenericUrl(UriTemplate.expand(
        arvadosRootUrl + restDescription.getBasePath() + method.getPath(), 
        parameters, true));

    try {
      // construct the request
      HttpRequestFactory requestFactory;
      requestFactory = httpTransport.createRequestFactory();

      // possibly required content
      HttpContent content = null;

      if (!method.getHttpMethod().equals("GET") && !method.getHttpMethod().equals("DELETE")) {
        String objectName = resourceName.substring(0, resourceName.length()-1);
        Object requestBody = paramsMap.get(objectName);
        if (requestBody == null) {
          error("POST method requires content object " + objectName);
        }
        
        content = new ByteArrayContent("application/json", ((String)requestBody).getBytes());
      }
      
      HttpRequest request = requestFactory.buildRequest(method.getHttpMethod(), url, content);

      // make the request
      List<String> authHeader = new ArrayList<String>();
      authHeader.add("OAuth2 " + arvadosApiToken);
      request.getHeaders().put("Authorization", authHeader);
      String response = request.execute().parseAsString();

      logger.debug(response);

      return response;
    } catch (Exception e) {
      e.printStackTrace();
      throw e;
    }
  }

  private HashMap<String, Object> loadParameters(Map<String, Object> paramsMap,
      RestMethod method) throws Exception {
    HashMap<String, Object> parameters = Maps.newHashMap();

    // required parameters
    if (method.getParameterOrder() != null) {
      for (String parameterName : method.getParameterOrder()) {
        JsonSchema parameter = method.getParameters().get(parameterName);
        if (Boolean.TRUE.equals(parameter.getRequired())) {
          Object parameterValue = paramsMap.get(parameterName);
          if (parameterValue == null) {
            error("missing required parameter: " + parameter);
          } else {
            putParameter(null, parameters, parameterName, parameter, parameterValue);
          }
        }
      }
    }

    for (Map.Entry<String, Object> entry : paramsMap.entrySet()) {
      String parameterName = entry.getKey();
      Object parameterValue = entry.getValue();
      
      if (parameterName.equals("contentType")) {
        if (method.getHttpMethod().equals("GET") || method.getHttpMethod().equals("DELETE")) {
          error("HTTP content type cannot be specified for this method: " + parameterName);
        }
      } else {
        JsonSchema parameter = null;
        if (restDescription.getParameters() != null) {
          parameter = restDescription.getParameters().get(parameterName);
        }
        if (parameter == null && method.getParameters() == null) {
          parameter = method.getParameters().get(parameterName);
        }
        putParameter(parameterName, parameters, parameterName, parameter, parameterValue);
      }
    }
    
    return parameters;
  }

  private RestMethod getMatchingMethod(String resourceName, String methodName)
              throws Exception {
    if (resourceName == null) {
      error("missing resource name");      
    }
    if (methodName == null) {
      error("missing method name");      
    }

    // initialize rest description if not already
    if (restDescription == null) {
      restDescription = loadArvadosApi();
    }

    Map<String, RestMethod> methodMap = null;
    Map<String, RestResource> resources = restDescription.getResources();
    RestResource resource = resources.get(resourceName);
    if (resource == null) {
      error("resource not found");
    }
    methodMap = resource.getMethods();
    RestMethod method =
        methodMap == null ? null : methodMap.get(methodName);
    if (method == null) {
      error("method not found: ");
    }
    
    return method;
  }

  /**
   * Not thread-safe. So, create for each request.
   * @param apiName
   * @param apiVersion
   * @return
   * @throws Exception
   */
  private RestDescription loadArvadosApi()
      throws Exception {
    try {
      Discovery discovery;

      Discovery.Builder discoveryBuilder = new Discovery.Builder(httpTransport, jsonFactory, null);

      discoveryBuilder.setRootUrl(arvadosRootUrl);
      discoveryBuilder.setApplicationName(apiName);

      discovery = discoveryBuilder.build();

      return discovery.apis().getRest(apiName, apiVersion).execute();
    } catch (Exception e) {
      e.printStackTrace();
      throw e;
    }
  }

  private void processMethods(
      ArrayList<MethodDetails> result, String resourceName, Map<String, RestMethod> methodMap) {
    if (methodMap == null) {
      return;
    }
    for (Map.Entry<String, RestMethod> methodEntry : methodMap.entrySet()) {
      MethodDetails details = new MethodDetails();
      String methodName = methodEntry.getKey();
      RestMethod method = methodEntry.getValue();
      details.name = (resourceName.isEmpty() ? "" : resourceName + ".") + methodName;
      details.hasContent =
          !method.getHttpMethod().equals("GET") && !method.getHttpMethod().equals("DELETE");
      // required parameters
      if (method.getParameterOrder() != null) {
        for (String parameterName : method.getParameterOrder()) {
          JsonSchema parameter = method.getParameters().get(parameterName);
          if (Boolean.TRUE.equals(parameter.getRequired())) {
            details.requiredParameters.add(parameterName);
          }
        }
      }
      // optional parameters
      Map<String, JsonSchema> parameters = method.getParameters();
      if (parameters != null) {
        for (Map.Entry<String, JsonSchema> parameterEntry : parameters.entrySet()) {
          String parameterName = parameterEntry.getKey();
          JsonSchema parameter = parameterEntry.getValue();
          if (!Boolean.TRUE.equals(parameter.getRequired())) {
            details.optionalParameters.add(parameterName);
          }
        }
      }
      result.add(details);
    }
  }

  private void processResources(
      ArrayList<MethodDetails> result, String resourceName, Map<String, RestResource> resourceMap) {
    if (resourceMap == null) {
      return;
    }
    for (Map.Entry<String, RestResource> entry : resourceMap.entrySet()) {
      RestResource resource = entry.getValue();
      String curResourceName = (resourceName.isEmpty() ? "" : resourceName + ".") + entry.getKey();
      processMethods(result, curResourceName, resource.getMethods());
      processResources(result, curResourceName, resource.getResources());
    }
  }

  private void putParameter(String argName, Map<String, Object> parameters,
      String parameterName, JsonSchema parameter, Object parameterValue) throws Exception {
    Object value = parameterValue;
    if (parameter != null) {
      if ("boolean".equals(parameter.getType())) {
        value = Boolean.valueOf(parameterValue.toString());
      } else if ("number".equals(parameter.getType())) {
        value = new BigDecimal(parameterValue.toString());
      } else if ("integer".equals(parameter.getType())) {
        value = new BigInteger(parameterValue.toString());
      }
    }
    
    parameters.put(parameterName, value);
  }

  private static void error(String detail) throws Exception {
    String errorDetail = "ERROR: " + detail;

    logger.debug(errorDetail);
    throw new Exception(errorDetail);
  }
  
  public static void main(String[] args){
    System.out.println("Welcome to Arvados Java SDK.");
    System.out.println("Please refer to README to learn to use the SDK.");
  }
}