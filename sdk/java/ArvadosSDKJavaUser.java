/**
 * This Sample test program is useful in getting started with working with Arvados Java SDK.
 * Please also see arvadso 
 * @author radhika
 *
 */

import java.io.File;
import java.io.BufferedWriter;
import java.io.FileWriter;

import java.util.ArrayList;
import java.util.List;

import org.arvados.sdk.java.Arvados;
import org.json.simple.JSONObject;
import org.json.simple.parser.JSONParser;

import com.google.api.services.discovery.model.RestDescription;

public class ArvadosSDKJavaUser {
  /** Make sure the following environment variables are set before using Arvados:
   *      ARVADOS_API_TOKEN, ARVADOS_API_HOST, ARVADOS_API_HOST_INSECURE
   */
  public static void main(String[] args) throws Exception {
    String apiName = "arvados";
    String apiVersion = "v1";

    Arvados arv = new Arvados(apiName);

    // Make a discover request. 
    System.out.println("Making an arvados discovery api request");
    List<String> params = new ArrayList<String>();
    params.add("discover");
    params.add("arvados");
    params.add("v1");

    RestDescription restDescription = arv.discover(params);
    System.out.println("Arvados discovery docuemnt:\n" + restDescription);
    
    // Make a users.list call
    System.out.println("Making an arvados users.list api call");

    params = new ArrayList<String>();
    params.add("call");
    params.add("arvados");
    params.add("v1");
    params.add("users.list");

    String response = arv.call(params);
    System.out.println("Arvados users.list:\n" + response);

    // get uuid of the first user from the response
    JSONParser parser = new JSONParser();
    Object obj = parser.parse(response);
    JSONObject jsonObject = (JSONObject) obj;
    List items = (List)jsonObject.get("items");

    JSONObject firstUser = (JSONObject)items.get(0);
    String userUuid = (String)firstUser.get("uuid");
    
    // Make a users.get call on the uuid obtained above
    System.out.println("Making a users.get for " + userUuid);
    params = new ArrayList<String>();
    params.add("call");
    params.add("arvados");
    params.add("v1");
    params.add("users.get");
    params.add(userUuid);
    response = arv.call(params);
    System.out.println("Arvados users.get:\n" + response);

    // Make a users.create call
    System.out.println("Making a users.create call.");
    
    File file = new File("/tmp/arvados_test.json");
    BufferedWriter output = new BufferedWriter(new FileWriter(file));
    output.write("{}");
    output.close();
    String filePath = file.getPath();

    params = new ArrayList<String>();
    params.add("call");
    params.add("arvados");
    params.add("v1");
    params.add("users.create");
    params.add(filePath);
    response = arv.call(params);
    System.out.println("Arvados users.create:\n" + response);

    // Make a pipeline_templates.list call
    System.out.println("Making a pipeline_templates.list call.");

    params = new ArrayList<String>();
    params.add("call");
    params.add("arvados");
    params.add("v1");
    params.add("pipeline_templates.list");
    response = arv.call(params);

    System.out.println("Arvados pipelinetempates.list:\n" + response);
  }
}
