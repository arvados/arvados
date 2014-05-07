/**
 * This Sample test program is useful in getting started with using Arvados Java SDK.
 * This program creates an Arvados instance using the configured environment variables.
 * It then provides a prompt to input method name and input parameters. 
 * The program them invokes the API server to execute the specified method.  
 * 
 * @author radhika
 */

import org.arvados.sdk.java.Arvados;

import java.io.File;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import java.util.Set;
import java.io.BufferedReader;
import java.io.InputStreamReader;

public class ArvadosSDKJavaExampleWithPrompt {
  /**
   * Make sure the following environment variables are set before using Arvados:
   * ARVADOS_API_TOKEN, ARVADOS_API_HOST and ARVADOS_API_HOST_INSECURE Set
   * ARVADOS_API_HOST_INSECURE to true if you are using self-singed certificates
   * in development and want to bypass certificate validations.
   * 
   * Please refer to http://doc.arvados.org/api/index.html for a complete list
   * of the available API methods.
   */
  public static void main(String[] args) throws Exception {
    String apiName = "arvados";
    String apiVersion = "v1";

    System.out.print("Welcome to Arvados Java SDK");
    System.out.println("You can use this example to call API methods");
    System.out.println("Enter ^C when you want to quit");

    // use configured env variables for API TOKEN, HOST and HOST_INSECURE
    Arvados arv = new Arvados(apiName, apiVersion);

    while (true) {
      // prompt for resource
      System.out.println("\n\nEnter Resource name (for example users): ");

      // read resource name
      BufferedReader in = new BufferedReader(new InputStreamReader(System.in));
      String resourceName = in.readLine().trim();

      // read method name
      System.out.println("\nEnter method name (for example get): ");
      String methodName = in.readLine().trim();

      // read method parameters
      System.out.println("\nEnter parameter name, value (for example uuid, uuid-value): ");
      Map paramsMap = new HashMap();
      String param = "";
      try {
        do {
          param = in.readLine();
          if (param.isEmpty())
            break;
          int index = param.indexOf(","); // first comma
          String paramName = param.substring(0, index);
          String paramValue = param.substring(index+1);
          System.out.println(paramName + " " + paramValue);
          paramsMap.put(paramName.trim(), paramValue.trim());

          System.out.println("\nEnter parameter name, value (for example uuid, uuid-value): ");
        } while (!param.isEmpty());
      } catch (Exception e) {
        System.out.println (e.getMessage());
        System.out.println ("\nStart over");
        continue;
      }

      // Make a "call" for the given resource name and method name
      try {
        System.out.println ("Making a call for " + resourceName + " " + methodName);
        Map response = arv.call(resourceName, methodName, paramsMap);
        
        Set<Entry<String,Object>> entrySet = (Set<Entry<String,Object>>)response.entrySet();
        for (Map.Entry<String, Object> entry : entrySet) {
          if ("items".equals(entry.getKey())) {
            List items = (List)entry.getValue();
            for (Object item : items) {
              System.out.println("    " + item);
            }            
          } else {
            System.out.println(entry.getKey() + " = " + entry.getValue());
          }
        }
      } catch (Exception e){
        System.out.println (e.getMessage());
        System.out.println ("\nStart over");
      }
    }
  }
}
