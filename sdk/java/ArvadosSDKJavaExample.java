/**
 * This Sample test program is useful in getting started with working with Arvados Java SDK.
 * @author radhika
 *
 */

import org.arvados.sdk.java.Arvados;

import java.io.File;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ArvadosSDKJavaExample {
  /** Make sure the following environment variables are set before using Arvados:
   *      ARVADOS_API_TOKEN, ARVADOS_API_HOST and ARVADOS_API_HOST_INSECURE 
   *      Set ARVADOS_API_HOST_INSECURE to true if you are using self-singed
   *      certificates in development and want to bypass certificate validations.
   *
   *  If you are not using env variables, you can pass them to Arvados constructor.
   *
   *  Please refer to http://doc.arvados.org/api/index.html for a complete list
   *      of the available API methods.
   */
  public static void main(String[] args) throws Exception {
    String apiName = "arvados";
    String apiVersion = "v1";

    Arvados arv = new Arvados(apiName, apiVersion);

    // Make a users list call. Here list on users is the method being invoked.
    // Expect a Map containing the list of users as the response.
    System.out.println("Making an arvados users.list api call");

    Map<String, Object> params = new HashMap<String, Object>();

    Map response = arv.call("users", "list", params);
    System.out.println("Arvados users.list:\n" + response);

    // get uuid of the first user from the response
    List items = (List)response.get("items");

    Map firstUser = (Map)items.get(0);
    String userUuid = (String)firstUser.get("uuid");
    
    // Make a users get call on the uuid obtained above
    System.out.println("\n\n\nMaking a users.get call for " + userUuid);
    params = new HashMap<String, Object>();
    params.put("uuid", userUuid);
    response = arv.call("users", "get", params);
    System.out.println("Arvados users.get:\n" + response);

    // Make a pipeline_templates list call
    System.out.println("\n\n\nMaking a pipeline_templates.list call.");

    params = new HashMap<String, Object>();
    response = arv.call("pipeline_templates", "list", params);

    System.out.println("Arvados pipelinetempates.list:\n" + response);
  }
}
