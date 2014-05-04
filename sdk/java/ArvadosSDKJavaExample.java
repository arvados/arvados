/**
 * This Sample test program is useful in getting started with working with Arvados Java SDK.
 * Please also see arvadso 
 * @author radhika
 *
 */

import org.arvados.sdk.java.Arvados;

import org.json.simple.JSONObject;
import org.json.simple.parser.JSONParser;

import java.io.File;
import java.io.BufferedWriter;
import java.io.FileWriter;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ArvadosSDKJavaExample {
  /** Make sure the following environment variables are set before using Arvados:
   *      ARVADOS_API_TOKEN, ARVADOS_API_HOST, ARVADOS_API_HOST_INSECURE
   */
  public static void main(String[] args) throws Exception {
    String apiName = "arvados";
    String apiVersion = "v1";

    Arvados arv = new Arvados(apiName, apiVersion);

    // Make a users.list call
    System.out.println("Making an arvados users.list api call");

    Map<String, Object> params = new HashMap<String, Object>();

    String response = arv.call("users", "list", params);
    System.out.println("Arvados users.list:\n" + response);

    // get uuid of the first user from the response
    JSONParser parser = new JSONParser();
    Object obj = parser.parse(response);
    JSONObject jsonObject = (JSONObject) obj;
    List items = (List)jsonObject.get("items");

    JSONObject firstUser = (JSONObject)items.get(0);
    String userUuid = (String)firstUser.get("uuid");
    
    // Make a users.get call on the uuid obtained above
    System.out.println("Making a users.get call for " + userUuid);
    params = new HashMap<String, Object>();
    params.put("uuid", userUuid);
    response = arv.call("users", "get", params);
    System.out.println("Arvados users.get:\n" + response);

    // Make a users.create call
    System.out.println("Making a users.create call.");
    
    params = new HashMap<String, Object>();
    params.put("user", "{}");
    response = arv.call("users", "create", params);
    System.out.println("Arvados users.create:\n" + response);

    // delete the newly created user
    parser = new JSONParser();
    obj = parser.parse(response);
    jsonObject = (JSONObject) obj;
    userUuid = (String)jsonObject.get("uuid");
    params = new HashMap<String, Object>();
    params.put("uuid", userUuid);
    response = arv.call("users", "delete", params);

    // Make a pipeline_templates.list call
    System.out.println("Making a pipeline_templates.list call.");

    params = new HashMap<String, Object>();
    response = arv.call("pipeline_templates", "list", params);

    System.out.println("Arvados pipelinetempates.list:\n" + response);
  }
}
