package org.arvados.sdk.java;

import com.google.api.client.http.javanet.*;
import com.google.api.client.http.FileContent;
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

import java.io.File;
import java.math.BigDecimal;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import org.apache.log4j.Logger;

public class Arvados {
  // HttpTransport and JsonFactory are thread-safe. So, use global instances.
  private HttpTransport HTTP_TRANSPORT;
  private final JsonFactory JSON_FACTORY = JacksonFactory.getDefaultInstance();

  private static final Pattern METHOD_PATTERN = Pattern.compile("((\\w+)\\.)*(\\w+)");

  private String ARVADOS_API_TOKEN;
  private String ARVADOS_API_HOST;
  private boolean ARVADOS_API_HOST_INSECURE;

  private String ARVADOS_ROOT_URL;

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
        ARVADOS_API_TOKEN = token;
      } else {
        ARVADOS_API_TOKEN = System.getenv().get("ARVADOS_API_TOKEN");
        if (ARVADOS_API_TOKEN == null) {
          throw new Exception("Missing environment variable: ARVADOS_API_TOKEN");
        }
      }

      if (host != null) {
        ARVADOS_API_HOST = host;
      } else {
        ARVADOS_API_HOST = System.getenv().get("ARVADOS_API_HOST");      
        if (ARVADOS_API_HOST == null) {
          throw new Exception("Missing environment variable: ARVADOS_API_HOST");
        }
      }
      ARVADOS_ROOT_URL = "https://" + ARVADOS_API_HOST;
      ARVADOS_ROOT_URL += (ARVADOS_API_HOST.endsWith("/")) ? "" : "/";

      if (hostInsecure != null) {
        ARVADOS_API_HOST_INSECURE = Boolean.valueOf(hostInsecure);
      } else {
        ARVADOS_API_HOST_INSECURE = "true".equals(System.getenv().get("ARVADOS_API_HOST_INSECURE")) ? true : false;
      }
      
      // Create HTTP_TRANSPORT object
      NetHttpTransport.Builder builder = new NetHttpTransport.Builder();
      if (ARVADOS_API_HOST_INSECURE) {
        builder.doNotValidateCertificate();
      }
      HTTP_TRANSPORT = builder.build();
    } catch (Throwable t) {
      t.printStackTrace();
    }
  }
  
  /**
   * Make a discover call and cache the response in-memory. Reload the document on each invocation.
   * @param params
   * @return
   * @throws Exception
   */
  public RestDescription discover() throws Exception {
    restDescription = loadArvadosApi(apiName, apiVersion);

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

  public String call(String resourceName, String methodName, List<String> callParams) throws Exception {
    if (resourceName == null) {
      error("missing resource name");      
    }
    if (methodName == null) {
      error("missing method name");      
    }
    
    String fullMethodName = callParams.get(3);
    Matcher m = METHOD_PATTERN.matcher(fullMethodName);
    if (!m.matches()) {
      error ("invalid method name: " + fullMethodName);
    }

    // initialize rest description if not already
    if (restDescription == null) {
      restDescription = loadArvadosApi(callParams.get(1), callParams.get(2));
    }

    Map<String, RestMethod> methodMap = null;
    int curIndex = 0;
    int nextIndex = fullMethodName.indexOf('.');
    if (nextIndex == -1) {
      methodMap = restDescription.getMethods();
    } else {
      Map<String, RestResource> resources = restDescription.getResources();
      while (true) {
        RestResource resource = resources.get(fullMethodName.substring(curIndex, nextIndex));
        if (resource == null) {
          break;
        }
        curIndex = nextIndex + 1;
        nextIndex = fullMethodName.indexOf(curIndex + 1, '.');
        if (nextIndex == -1) {
          methodMap = resource.getMethods();
          break;
        }
        resources = resource.getResources();
      }
    }

    RestMethod method =
        methodMap == null ? null : methodMap.get(fullMethodName.substring(curIndex));
    if (method == null) {
      error("method not found: " + fullMethodName);
    }

    HashMap<String, Object> parameters = Maps.newHashMap();
    File requestBodyFile = null;
    String contentType = "application/json";

    // Start looking for params at index 4. The first 4 were: call arvados v1 <method_name>
    int i = 4;
    // required parameters
    if (method.getParameterOrder() != null) {
      for (String parameterName : method.getParameterOrder()) {
        JsonSchema parameter = method.getParameters().get(parameterName);
        if (Boolean.TRUE.equals(parameter.getRequired())) {
          if (i == callParams.size()) {
            error("missing required parameter: " + parameter);
          } else {
            putParameter(null, parameters, parameterName, parameter, callParams.get(i++));
          }
        }
      }
    }

    // possibly required content
    if (!method.getHttpMethod().equals("GET") && !method.getHttpMethod().equals("DELETE")) {
      String fileName = callParams.get(i++);
      requestBodyFile = new File(fileName);
      if (!requestBodyFile.canRead()) {
        error("POST method requires input file. Unable to read file: " + fileName);
      }
    }

    while (i < callParams.size()) {
      String argName = callParams.get(i++);
      if (!argName.startsWith("--")) {
        error("optional parameters must start with \"--\": " + argName);
      }
      String parameterName = argName.substring(2);
      if (i == callParams.size()) {
        error("missing parameter value for: " + argName);
      }
      String parameterValue = callParams.get(i++);
      if (parameterName.equals("contentType")) {
        contentType = parameterValue;
        if (method.getHttpMethod().equals("GET") || method.getHttpMethod().equals("DELETE")) {
          error("HTTP content type cannot be specified for this method: " + argName);
        }
      } else {
        JsonSchema parameter = null;
        if (restDescription.getParameters() != null) {
          parameter = restDescription.getParameters().get(parameterName);
        }
        if (parameter == null && method.getParameters() == null) {
          parameter = method.getParameters().get(parameterName);
        }
        putParameter(argName, parameters, parameterName, parameter, parameterValue);
      }
    }

    GenericUrl url = new GenericUrl(UriTemplate.expand(
        ARVADOS_ROOT_URL + restDescription.getBasePath() + method.getPath(), parameters,
        true));

    HttpContent content = null;
    if (requestBodyFile != null) {
      content = new FileContent(contentType, requestBodyFile);
    }

    try {
      HttpRequestFactory requestFactory;
      requestFactory = HTTP_TRANSPORT.createRequestFactory();

      HttpRequest request = requestFactory.buildRequest(method.getHttpMethod(), url, content);

      List<String> authHeader = new ArrayList<String>();
      authHeader.add("OAuth2 " + ARVADOS_API_TOKEN);
      request.getHeaders().put("Authorization", authHeader);
      String response = request.execute().parseAsString();

      logger.debug(response);
      
      return response;
    } catch (Exception e) {
      e.printStackTrace();
      throw e;
    }
  }

  /**
   * Not thread-safe. So, create for each request.
   * @param apiName
   * @param apiVersion
   * @return
   * @throws Exception
   */
  private RestDescription loadArvadosApi(String apiName, String apiVersion)
      throws Exception {
    try {
      Discovery discovery;
      
      Discovery.Builder discoveryBuilder = new Discovery.Builder(HTTP_TRANSPORT, JSON_FACTORY, null);

      discoveryBuilder.setRootUrl(ARVADOS_ROOT_URL);
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
      String parameterName, JsonSchema parameter, String parameterValue) throws Exception {
    Object value = parameterValue;
    if (parameter != null) {
      if ("boolean".equals(parameter.getType())) {
        value = Boolean.valueOf(parameterValue);
      } else if ("number".equals(parameter.getType())) {
        value = new BigDecimal(parameterValue);
      } else if ("integer".equals(parameter.getType())) {
        value = new BigInteger(parameterValue);
      }
    }
    Object oldValue = parameters.put(parameterName, value);
    if (oldValue != null) {
      error("duplicate parameter: " + argName);
    }
  }
  
  private static void error(String detail) throws Exception {
    String errorDetail = "ERROR: " + detail;
    
    logger.debug(errorDetail);
    throw new Exception(errorDetail);
  }

}
