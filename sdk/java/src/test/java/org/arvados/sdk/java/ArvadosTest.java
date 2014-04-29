package org.arvados.sdk.java;

import java.io.BufferedWriter;
import java.io.File;
import java.io.FileWriter;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import junit.framework.Test;
import junit.framework.TestCase;
import junit.framework.TestSuite;

import com.google.api.services.discovery.model.RestDescription;
import com.google.api.services.discovery.model.RestResource;

import org.json.simple.JSONObject;
import org.json.simple.parser.JSONParser;

/**
 * Unit test for Arvados.
 */
public class ArvadosTest extends TestCase {
  /**
   * Create the test case
   *
   * @param testName name of the test case
   */
  public ArvadosTest(String testName) {
    super( testName );
  }

  public static Test suite() {
    return new TestSuite(ArvadosTest.class);
  }

  public void testShowMainHelp() {
    Arvados.showMainHelp();
  }

  /**
   * test discover method
   * @throws Exception
   */
  public void testDiscover() throws Exception {
    Arvados arv = new Arvados();

    List<String> params = new ArrayList<String>();
    params.add("discover");
    params.add("arvados");
    params.add("v1");

    RestDescription restDescription = arv.discover(params);

    // The discover method returns the supported methods
    Map<String, RestResource> resources = restDescription.getResources();
    assertNotNull("Expected resources", resources);
    //assertNotNull("Expected methods", restDescription.getMethods());

    Object users = resources.get("users");
    assertNotNull ("Expected users.list method", users);
    assertEquals("Exepcted users.list to be a RestResource type", RestResource.class, users.getClass());

    assertTrue("Root URL expected to match ARVADOS_API_HOST env paramdeter", 
        restDescription.getRootUrl().contains(System.getenv().get("ARVADOS_API_HOST")));
  }

  /**
   * Test users.list api
   * @throws Exception
   */
  public void testCallUsersList() throws Exception {
    Arvados arv = new Arvados();

    List<String> callParams = new ArrayList<String>();
    callParams.add("call");
    callParams.add("arvados");
    callParams.add("v1");
    callParams.add("users.list");

    String callResponse = arv.call(callParams);
    assertTrue("Expected users.list in response", callResponse.contains("arvados#userList"));
    assertTrue("Expected users.list in response", callResponse.contains("uuid"));

    JSONParser parser = new JSONParser();
    Object obj = parser.parse(callResponse);
    JSONObject jsonObject = (JSONObject) obj;

    assertEquals("Expected kind to be users.list", "arvados#userList", jsonObject.get("kind"));

    List items = (List)jsonObject.get("items");
    assertNotNull("expected users list items", items);
    assertTrue("expected at least one item in users list", items.size()>0);

    JSONObject firstUser = (JSONObject)items.get(0);
    assertNotNull ("Expcted at least one user", firstUser);

    assertEquals("Expected kind to be user", "arvados#user", firstUser.get("kind"));
    assertNotNull("Expected uuid for first user", firstUser.get("uuid"));
  }

  /**
   * Test users.get <uuid> api
   * @throws Exception
   */
  public void testCallUsersGet() throws Exception {
    Arvados arv = new Arvados();

    // call user.system and get uuid of this user
    List<String> callParams = new ArrayList<String>();
    callParams.add("call");
    callParams.add("arvados");
    callParams.add("v1");
    callParams.add("users.list");

    String callResponse = arv.call(callParams);
    JSONParser parser = new JSONParser();
    Object obj = parser.parse(callResponse);
    JSONObject jsonObject = (JSONObject) obj;
    assertNotNull("expected users list", jsonObject);
    List items = (List)jsonObject.get("items");
    assertNotNull("expected users list items", items);

    JSONObject firstUser = (JSONObject)items.get(0);
    String userUuid = (String)firstUser.get("uuid");

    // invoke users.get with the system user uuid
    callParams = new ArrayList<String>();
    callParams.add("call");
    callParams.add("arvados");
    callParams.add("v1");
    callParams.add("users.get");
    callParams.add(userUuid);

    callResponse = arv.call(callParams);

    //JSONParser parser = new JSONParser();
    jsonObject = (JSONObject) parser.parse(callResponse);;
    assertNotNull("Expected uuid for first user", jsonObject.get("uuid"));
    assertEquals("Expected system user uuid", userUuid, jsonObject.get("uuid"));
  }

  /**
   * Test users.create api
   * @throws Exception
   */
  //@Ignore
  public void testCreateUser() throws Exception {
    Arvados arv = new Arvados();

    // POST request needs an input file
    File file = new File("/tmp/arvados_test.json");
    BufferedWriter output = new BufferedWriter(new FileWriter(file));
    output.write("{}");
    output.close();

    List<String> callParams = new ArrayList<String>();
    callParams.add("call");
    callParams.add("arvados");
    callParams.add("v1");
    callParams.add("users.create");
    callParams.add("/tmp/arvados_test.json");
    String callResponse = arv.call(callParams);

    JSONParser parser = new JSONParser();
    JSONObject jsonObject = (JSONObject) parser.parse(callResponse);
    assertEquals("Expected kind to be user", "arvados#user", jsonObject.get("kind"));
    assertNotNull("Expected uuid for first user", jsonObject.get("uuid"));
    
    file.delete();
  }
}

