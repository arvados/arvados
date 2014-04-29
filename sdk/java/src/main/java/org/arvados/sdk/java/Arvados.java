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
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

public class Arvados {
  private static HttpTransport HTTP_TRANSPORT;
  private static final JsonFactory JSON_FACTORY = JacksonFactory.getDefaultInstance();
  private static Discovery DISCOVERY_CLIENT;

  private static final Pattern METHOD_PATTERN = Pattern.compile("((\\w+)\\.)*(\\w+)");

  private static String ARVADOS_API_TOKEN;
  private static String ARVADOS_API_HOST;
  private static String ARVADOS_API_HOST_INSECURE;

  private static String ARVADOS_ROOT_URL;

  public static void main(String[] args) throws Exception {
    if (args.length == 0) {
      showMainHelp();
        error(null, "Missing input args");
    } else {
      String command = args[0];
      if (command.equals("help")) {
        help(args);
      } else if (command.equals("call")) {
        List<String> params = Arrays.asList(args);
        
        if (args.length == 1) {
          error("call", "missing api name");
        }
        
        Arvados arv = new Arvados(args[1]);
        String response = arv.call(params);
        System.out.println (response);
      } else if (command.equals("discover")) {
        List<String> params = Arrays.asList(args);
        
        if (args.length == 1) {
          error("call", "missing api name");
        }
        
        Arvados arv = new Arvados(args[1]);
        arv.discover(params);
      } else {
        error(null, "unknown command: " + command);
      }
    }
  }

  protected static void help(String[] args) throws Exception {
    if (args.length == 1) {
      showMainHelp();
    } else {
      String helpCommand = args[1];
      if (helpCommand.equals("call")) {
        System.out.println("Usage: Arvados call methodName [parameters]");
        System.out.println();
        System.out.println("Examples:");
        System.out.println("  Arvados call arvados v1 users.list");
        System.out.println("  Arvados call arvados v1 users.get <uuid>");
        System.out.println("  Arvados call arvados v1 pipeline_instances.list");
      } else if (helpCommand.equals("discover")) {
        System.out.println("Usage");
        System.out.println("Examples:");
        System.out.println("  Arvados discover arvados v1");
      } else {
        error(null, "unknown command: " + helpCommand);
      }
    }
  }

  protected static void showMainHelp() {
    System.out.println("arvados");
    System.out.println();
    System.out.println("For more help on a specific command, type one of:");
    System.out.println();
    System.out.println("  Arvados help call");
    System.out.println("  Arvados help discover");
  }

  private static void error(String command, String detail) throws Exception {
    String errorDetail = "ERROR: " + detail +
        "For help, type: Arvados" + (command == null ? "" : " help " + command);
    
    throw new Exception(errorDetail);
  }

  public Arvados (String apiName){
    try {
      // Read needed environmental variables
      ARVADOS_API_TOKEN = System.getenv().get("ARVADOS_API_TOKEN");
      if (ARVADOS_API_TOKEN == null) {
        throw new Exception("Missing environment variable: ARVADOS_API_TOKEN");
      }

      ARVADOS_API_HOST = System.getenv().get("ARVADOS_API_HOST");      
      if (ARVADOS_API_HOST == null) {
        throw new Exception("Missing environment variable: ARVADOS_API_HOST");
      }

      ARVADOS_ROOT_URL = "https://" + ARVADOS_API_HOST;
      ARVADOS_ROOT_URL += (ARVADOS_API_HOST.endsWith("/")) ? "" : "/";

      ARVADOS_API_HOST_INSECURE = System.getenv().get("ARVADOS_API_HOST_INSECURE");
      if (ARVADOS_API_HOST_INSECURE == null) {
        throw new Exception("Missing environment variable: ARVADOS_API_HOST_INSECURE");
      }

      // Create HTTP_TRANSPORT object
      NetHttpTransport.Builder builder = new NetHttpTransport.Builder();
      builder.doNotValidateCertificate();
      HTTP_TRANSPORT = builder.build();

      // Create DISCOVERY_CLIENT object
      Discovery.Builder discoveryBuilder = new Discovery.Builder(HTTP_TRANSPORT, JSON_FACTORY, null);

      discoveryBuilder.setRootUrl(ARVADOS_ROOT_URL);
      discoveryBuilder.setApplicationName(apiName);
      DISCOVERY_CLIENT = discoveryBuilder.build();
    } catch (Throwable t) {
      t.printStackTrace();
    }
  }
  public String call(List<String> callParams) throws Exception {
    if (callParams.size() == 1) {
      error("call", "missing api name");
    } else if (callParams.size() == 2) {
      error("call", "missing api version");
    } else if (callParams.size() == 3) {
      error("call", "missing method name");
    }

    String fullMethodName = callParams.get(3);
    Matcher m = METHOD_PATTERN.matcher(fullMethodName);
    if (!m.matches()) {
      error("call", "invalid method name: " + fullMethodName);
    }

    RestDescription restDescription = loadArvadosApi(callParams.get(1), callParams.get(2));
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
      error("call", "method not found: " + fullMethodName);
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
            error("call", "missing required parameter: " + parameter);
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
        error("call", "POST method requires input file. Unable to read file: " + fileName);
      }
    }

    while (i < callParams.size()) {
      String argName = callParams.get(i++);
      if (!argName.startsWith("--")) {
        error("call", "optional parameters must start with \"--\": " + argName);
      }
      String parameterName = argName.substring(2);
      if (i == callParams.size()) {
        error("call", "missing parameter value for: " + argName);
      }
      String parameterValue = callParams.get(i++);
      if (parameterName.equals("contentType")) {
        contentType = parameterValue;
        if (method.getHttpMethod().equals("GET") || method.getHttpMethod().equals("DELETE")) {
          error("call", "HTTP content type cannot be specified for this method: " + argName);
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
      //request.getp
      List<String> authHeader = new ArrayList<String>();
      authHeader.add("OAuth2 " + ARVADOS_API_TOKEN);
      request.getHeaders().put("Authorization", authHeader);
      String response = request.execute().parseAsString();

      return response;
    } catch (Exception e) {
      e.printStackTrace();
      throw e;
    }
  }

  public RestDescription discover(List<String> params) throws Exception {
    if (params.size() == 1) {
      error("call", "missing api name");
    } else if (params.size() == 2) {
      error("call", "missing api version");
    } 

    RestDescription restDescription = loadArvadosApi(params.get(1), params.get(2));

    // compute method details
    ArrayList<MethodDetails> result = Lists.newArrayList();
    String resourceName = "";
    processResources(result, resourceName, restDescription.getResources());


    // display method details
    Collections.sort(result);
    for (MethodDetails methodDetail : result) {
      System.out.println();
      System.out.print("Arvados call " + params.get(1) + " " + params.get(2) + " " + methodDetail.name);
      for (String param : methodDetail.requiredParameters) {
        System.out.print(" <" + param + ">");
      }
      if (methodDetail.hasContent) {
        System.out.print(" contentFile");
      }
      if (methodDetail.optionalParameters.isEmpty() && !methodDetail.hasContent) {
        System.out.println();
      } else {
        System.out.println(" [optional parameters...]");
        System.out.println("  --contentType <value> (default is \"application/json\")");
        for (String param : methodDetail.optionalParameters) {
          System.out.println("  --" + param + " <value>");
        }
      }
    }

    return (restDescription);
  }

  private RestDescription loadArvadosApi(String appName, String appVersion)
      throws Exception {
    try {
      return DISCOVERY_CLIENT.apis().getRest(appName, appVersion).execute();
    } catch (Exception e) {
      e.printStackTrace();
      throw e;
    }
  }

  private static void processMethods(
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

  private static void processResources(
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

  private static void putParameter(String argName, Map<String, Object> parameters,
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
      error("call", "duplicate parameter: " + argName);
    }
  }

}