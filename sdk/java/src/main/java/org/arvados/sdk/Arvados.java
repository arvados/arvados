package org.arvados.sdk;

import com.google.api.client.http.javanet.*;
import com.google.api.client.http.ByteArrayContent;
import com.google.api.client.http.GenericUrl;
import com.google.api.client.http.HttpBackOffIOExceptionHandler;
import com.google.api.client.http.HttpContent;
import com.google.api.client.http.HttpRequest;
import com.google.api.client.http.HttpRequestFactory;
import com.google.api.client.http.HttpTransport;
import com.google.api.client.http.UriTemplate;
import com.google.api.client.json.JsonFactory;
import com.google.api.client.json.jackson2.JacksonFactory;
import com.google.api.client.util.ExponentialBackOff;
import com.google.api.client.util.Maps;
import com.google.api.services.discovery.Discovery;
import com.google.api.services.discovery.model.JsonSchema;
import com.google.api.services.discovery.model.RestDescription;
import com.google.api.services.discovery.model.RestMethod;
import com.google.api.services.discovery.model.RestMethod.Request;
import com.google.api.services.discovery.model.RestResource;

import java.math.BigDecimal;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;

import org.apache.log4j.Logger;
import org.json.simple.JSONArray;
import org.json.simple.JSONObject;

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

  // Get it once and reuse on the call requests
  RestDescription restDescription = null;
  String apiName = null;
  String apiVersion = null;

  public Arvados (String apiName, String apiVersion) throws Exception {
    this (apiName, apiVersion, null, null, null);
  }

  public Arvados (String apiName, String apiVersion, String token,
      String host, String hostInsecure) throws Exception {
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
      arvadosApiHostInsecure =
          "true".equals(System.getenv().get("ARVADOS_API_HOST_INSECURE")) ? true : false;
    }

    // Create HTTP_TRANSPORT object
    NetHttpTransport.Builder builder = new NetHttpTransport.Builder();
    if (arvadosApiHostInsecure) {
      builder.doNotValidateCertificate();
    }
    httpTransport = builder.build();

    // initialize rest description
    restDescription = loadArvadosApi();
  }

  /**
   * Make a call to API server with the provide call information.
   * @param resourceName
   * @param methodName
   * @param paramsMap
   * @return Map
   * @throws Exception
   */
  public Map call(String resourceName, String methodName,
      Map<String, Object> paramsMap) throws Exception {
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

      if (!method.getHttpMethod().equals("GET") &&
          !method.getHttpMethod().equals("DELETE")) {
        String objectName = resourceName.substring(0, resourceName.length()-1);
        Object requestBody = paramsMap.get(objectName);
        if (requestBody == null) {
          error("POST method requires content object " + objectName);
        }

        content = new ByteArrayContent("application/json",((String)requestBody).getBytes());
      }

      HttpRequest request =
          requestFactory.buildRequest(method.getHttpMethod(), url, content);

      // Set read timeout to 120 seconds (up from default of 20 seconds)
      request.setReadTimeout(120 * 1000);

      // Add retry behavior
      request.setIOExceptionHandler(new HttpBackOffIOExceptionHandler(new ExponentialBackOff()));

      // make the request
      List<String> authHeader = new ArrayList<String>();
      authHeader.add("OAuth2 " + arvadosApiToken);
      request.getHeaders().put("Authorization", authHeader);
      String response = request.execute().parseAsString();

      Map responseMap = jsonFactory.createJsonParser(response).parse(HashMap.class);

      logger.debug(responseMap);

      return responseMap;
    } catch (Exception e) {
      e.printStackTrace();
      throw e;
    }
  }

  /**
   * Get all supported resources by the API
   * @return Set
   */
  public Set<String> getAvailableResourses() {
    return (restDescription.getResources().keySet());
  }

  /**
   * Get all supported method names for the given resource
   * @param resourceName
   * @return Set
   * @throws Exception
   */
  public Set<String> getAvailableMethodsForResourse(String resourceName)
      throws Exception {
    Map<String, RestMethod> methodMap = getMatchingMethodMap (resourceName);
    return (methodMap.keySet());
  }

  /**
   * Get the parameters for the method in the resource sought.
   * @param resourceName
   * @param methodName
   * @return Set
   * @throws Exception
   */
  public Map<String,List<String>> getAvailableParametersForMethod(String resourceName, String methodName)
      throws Exception {
    RestMethod method = getMatchingMethod(resourceName, methodName);
    Map<String, List<String>> parameters = new HashMap<String, List<String>>();
    List<String> requiredParameters = new ArrayList<String>();
    List<String> optionalParameters = new ArrayList<String>();
    parameters.put ("required", requiredParameters);
    parameters.put("optional", optionalParameters);

    try {
      // get any request parameters
      Request request = method.getRequest();
      if (request != null) {
        Object required = request.get("required");
        Object requestProperties = request.get("properties");
        if (requestProperties != null) {
          if (requestProperties instanceof Map) {
            Map properties = (Map)requestProperties;
            Set<String> propertyKeys = properties.keySet();
            for (String property : propertyKeys) {
              if (Boolean.TRUE.equals(required)) {
                requiredParameters.add(property);
              } else {
                optionalParameters.add(property);
              }
            }
          }
        }
      }

      // get other listed parameters
      Map<String,JsonSchema> methodParameters = method.getParameters();
      for (Map.Entry<String, JsonSchema> entry : methodParameters.entrySet()) {
        if (Boolean.TRUE.equals(entry.getValue().getRequired())) {
          requiredParameters.add(entry.getKey());
        } else {
          optionalParameters.add(entry.getKey());
        }
      }
    } catch (Exception e){
      logger.error(e);
    }

    return parameters;
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
        if (parameter == null && method.getParameters() != null) {
          parameter = method.getParameters().get(parameterName);
        }
        putParameter(parameterName, parameters, parameterName, parameter, parameterValue);
      }
    }

    return parameters;
  }

  private RestMethod getMatchingMethod(String resourceName, String methodName)
      throws Exception {
    Map<String, RestMethod> methodMap = getMatchingMethodMap(resourceName);

    if (methodName == null) {
      error("missing method name");
    }

    RestMethod method =
        methodMap == null ? null : methodMap.get(methodName);
    if (method == null) {
      error("method not found: ");
    }

    return method;
  }

  private Map<String, RestMethod> getMatchingMethodMap(String resourceName)
      throws Exception {
    if (resourceName == null) {
      error("missing resource name");
    }

    Map<String, RestMethod> methodMap = null;
    Map<String, RestResource> resources = restDescription.getResources();
    RestResource resource = resources.get(resourceName);
    if (resource == null) {
      error("resource not found");
    }
    methodMap = resource.getMethods();
    return methodMap;
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

      Discovery.Builder discoveryBuilder =
          new Discovery.Builder(httpTransport, jsonFactory, null);

      discoveryBuilder.setRootUrl(arvadosRootUrl);
      discoveryBuilder.setApplicationName(apiName);

      discovery = discoveryBuilder.build();

      return discovery.apis().getRest(apiName, apiVersion).execute();
    } catch (Exception e) {
      e.printStackTrace();
      throw e;
    }
  }

  /**
   * Convert the input parameter into its equivalent json string.
   * Add this json string value to the parameters map to be sent to server.
   * @param argName
   * @param parameters
   * @param parameterName
   * @param parameter
   * @param parameterValue
   * @throws Exception
   */
  private void putParameter(String argName, Map<String, Object> parameters,
      String parameterName, JsonSchema parameter, Object parameterValue)
          throws Exception {
    Object value = parameterValue;
    if (parameter != null) {
      if ("boolean".equals(parameter.getType())) {
        value = Boolean.valueOf(parameterValue.toString());
      } else if ("number".equals(parameter.getType())) {
        value = new BigDecimal(parameterValue.toString());
      } else if ("integer".equals(parameter.getType())) {
        value = new BigInteger(parameterValue.toString());
      } else if ("float".equals(parameter.getType())) {
        value = new BigDecimal(parameterValue.toString());
      } else if ("Java.util.Calendar".equals(parameter.getType())) {
        value = new BigDecimal(parameterValue.toString());
      } else if (("array".equals(parameter.getType())) ||
          ("Array".equals(parameter.getType()))) {
        if (parameterValue.getClass().isArray()){
          value = getJsonValueFromArrayType(parameterValue);
        } else if (List.class.isAssignableFrom(parameterValue.getClass())) {
          value = getJsonValueFromListType(parameterValue);
        }
      } else if (("Hash".equals(parameter.getType())) ||
          ("hash".equals(parameter.getType()))) {
        value = getJsonValueFromMapType(parameterValue);
      } else {
        if (parameterValue.getClass().isArray()){
          value = getJsonValueFromArrayType(parameterValue);
        } else if (List.class.isAssignableFrom(parameterValue.getClass())) {
          value = getJsonValueFromListType(parameterValue);
        } else if (Map.class.isAssignableFrom(parameterValue.getClass())) {
          value = getJsonValueFromMapType(parameterValue);
        }
      }
    }

    parameters.put(parameterName, value);
  }

  /**
   * Convert the given input array into json string before sending to server.
   * @param parameterValue
   * @return
   */
  private String getJsonValueFromArrayType (Object parameterValue) {
    String arrayStr = Arrays.deepToString((Object[])parameterValue);

    // we can expect either an array of array objects or an array of objects
    if (arrayStr.startsWith("[[") && arrayStr.endsWith("]]")) {
      Object[][] array = new Object[1][];
      arrayStr = arrayStr.substring(2, arrayStr.length()-2);
      String jsonStr = getJsonStringForArrayStr(arrayStr);
      String value = "[" + jsonStr + "]";
      return value;
    } else {
      arrayStr = arrayStr.substring(1, arrayStr.length()-1);
      return (getJsonStringForArrayStr(arrayStr));
    }
  }

  private String getJsonStringForArrayStr(String arrayStr) {
    Object[] array = arrayStr.split(",");
    Object[] trimmedArray = new Object[array.length];
    for (int i=0; i<array.length; i++){
      trimmedArray[i] = array[i].toString().trim();
    }
    String value = JSONArray.toJSONString(Arrays.asList(trimmedArray));
    return value;
  }

  /**
   * Convert the given input List into json string before sending to server.
   * @param parameterValue
   * @return
   */
  private String getJsonValueFromListType (Object parameterValue) {
    List paramList = (List)parameterValue;
    Object[] array = new Object[paramList.size()];
    Arrays.deepToString(paramList.toArray(array));
    return (getJsonValueFromArrayType(array));
  }

  /**
   * Convert the given input map into json string before sending to server.
   * @param parameterValue
   * @return
   */
  private String getJsonValueFromMapType (Object parameterValue) {
    JSONObject json = new JSONObject((Map)parameterValue);
    return json.toString();
  }

  private static void error(String detail) throws Exception {
    String errorDetail = "ERROR: " + detail;

    logger.debug(errorDetail);
    throw new Exception(errorDetail);
  }

  public static void main(String[] args){
    System.out.println("Welcome to Arvados Java SDK.");
    System.out.println("Please refer to http://doc.arvados.org/sdk/java/index.html to get started with the the SDK.");
  }

}
