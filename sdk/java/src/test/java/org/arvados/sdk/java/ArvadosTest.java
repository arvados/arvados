package org.arvados.sdk.java;

import java.io.File;
import java.io.FileInputStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import org.junit.Test;

import static org.junit.Assert.*;

import org.json.simple.JSONObject;
import org.json.simple.parser.JSONParser;

/**
 * Unit test for Arvados.
 */
public class ArvadosTest {

  /**
   * Test users.list api
   * @throws Exception
   */
  @Test
  public void testCallUsersList() throws Exception {
    Arvados arv = new Arvados("arvados", "v1");

    Map<String, Object> params = new HashMap<String, Object>();

    String response = arv.call("users", "list", params);
    assertTrue("Expected users.list in response", response.contains("arvados#userList"));
    assertTrue("Expected users.list in response", response.contains("uuid"));

    JSONParser parser = new JSONParser();
    Object obj = parser.parse(response);
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
  @Test
  public void testCallUsersGet() throws Exception {
    Arvados arv = new Arvados("arvados", "v1");

    // call user.system and get uuid of this user
    Map<String, Object> params = new HashMap<String, Object>();

    String response = arv.call("users", "list", params);
    JSONParser parser = new JSONParser();
    Object obj = parser.parse(response);
    JSONObject jsonObject = (JSONObject) obj;
    assertNotNull("expected users list", jsonObject);
    List items = (List)jsonObject.get("items");
    assertNotNull("expected users list items", items);

    JSONObject firstUser = (JSONObject)items.get(0);
    String userUuid = (String)firstUser.get("uuid");

    // invoke users.get with the system user uuid
    params = new HashMap<String, Object>();
    params.put("uuid", userUuid);

    response = arv.call("users", "get", params);

    //JSONParser parser = new JSONParser();
    jsonObject = (JSONObject) parser.parse(response);;
    assertNotNull("Expected uuid for first user", jsonObject.get("uuid"));
    assertEquals("Expected system user uuid", userUuid, jsonObject.get("uuid"));
  }

  /**
   * Test users.create api
   * @throws Exception
   */
  @Test
  public void testCreateUser() throws Exception {
    Arvados arv = new Arvados("arvados", "v1");

    Map<String, Object> params = new HashMap<String, Object>();
    params.put("user", "{}");
    String response = arv.call("users", "create", params);

    JSONParser parser = new JSONParser();
    JSONObject jsonObject = (JSONObject) parser.parse(response);
    assertEquals("Expected kind to be user", "arvados#user", jsonObject.get("kind"));

    Object uuid = jsonObject.get("uuid");
    assertNotNull("Expected uuid for first user", uuid);

    // delete the object
    params = new HashMap<String, Object>();
    params.put("uuid", uuid);
    response = arv.call("users", "delete", params);

    // invoke users.get with the system user uuid
    params = new HashMap<String, Object>();
    params.put("uuid", uuid);

    Exception caught = null;
    try {
      arv.call("users", "get", params);
    } catch (Exception e) {
      caught = e;
    }

    assertNotNull ("expected exception", caught);
    assertTrue ("Expected 404", caught.getMessage().contains("Path not found"));
  }

  @Test
  public void testCreateUserWithMissingRequiredParam() throws Exception {
    Arvados arv = new Arvados("arvados", "v1");

    Map<String, Object> params = new HashMap<String, Object>();

    Exception caught = null;
    try {
      arv.call("users", "create", params);
    } catch (Exception e) {
      caught = e;
    }

    assertNotNull ("expected exception", caught);
    assertTrue ("Expected POST method requires content object user", 
        caught.getMessage().contains("ERROR: POST method requires content object user"));
  }

  /**
   * Test users.create api
   * @throws Exception
   */
  @Test
  public void testCreateAndUpdateUser() throws Exception {
    Arvados arv = new Arvados("arvados", "v1");

    Map<String, Object> params = new HashMap<String, Object>();
    params.put("user", "{}");
    String response = arv.call("users", "create", params);

    JSONParser parser = new JSONParser();
    JSONObject jsonObject = (JSONObject) parser.parse(response);
    assertEquals("Expected kind to be user", "arvados#user", jsonObject.get("kind"));

    Object uuid = jsonObject.get("uuid");
    assertNotNull("Expected uuid for first user", uuid);

    // update this user
    params = new HashMap<String, Object>();
    params.put("user", "{}");
    params.put("uuid", uuid);
    response = arv.call("users", "update", params);

    parser = new JSONParser();
    jsonObject = (JSONObject) parser.parse(response);
    assertEquals("Expected kind to be user", "arvados#user", jsonObject.get("kind"));

    uuid = jsonObject.get("uuid");
    assertNotNull("Expected uuid for first user", uuid);

    // delete the object
    params = new HashMap<String, Object>();
    params.put("uuid", uuid);
    response = arv.call("users", "delete", params);
  }

  /**
   * Test unsupported api version api
   * @throws Exception
   */
  @Test
  public void testUnsupportedApiName() throws Exception {
    Arvados arv = new Arvados("not_arvados", "v1");

    Exception caught = null;
    try {
      arv.call("users", "list", null);
    } catch (Exception e) {
      caught = e;
    }

    assertNotNull ("expected exception", caught);
    assertTrue ("Expected 404 when unsupported api is used", caught.getMessage().contains("404 Not Found"));
  }

  /**
   * Test unsupported api version api
   * @throws Exception
   */
  @Test
  public void testUnsupportedVersion() throws Exception {
    Arvados arv = new Arvados("arvados", "v2");

    Exception caught = null;
    try {
      arv.call("users", "list", null);
    } catch (Exception e) {
      caught = e;
    }

    assertNotNull ("expected exception", caught);
    assertTrue ("Expected 404 when unsupported version is used", caught.getMessage().contains("404 Not Found"));
  }

  /**
   * Test unsupported api version api
   * @throws Exception
   */
  @Test
  public void testCallForNoSuchResrouce() throws Exception {
    Arvados arv = new Arvados("arvados", "v1");

    Exception caught = null;
    try {
      arv.call("abcd", "list", null);
    } catch (Exception e) {
      caught = e;
    }

    assertNotNull ("expected exception", caught);
    assertTrue ("Expected ERROR: 404 not found", caught.getMessage().contains("ERROR: resource not found"));
  }

  /**
   * Test unsupported api version api
   * @throws Exception
   */
  @Test
  public void testCallForNoSuchResrouceMethod() throws Exception {
    Arvados arv = new Arvados("arvados", "v1");

    Exception caught = null;
    try {
      arv.call("users", "abcd", null);
    } catch (Exception e) {
      caught = e;
    }

    assertNotNull ("expected exception", caught);
    assertTrue ("Expected ERROR: 404 not found", caught.getMessage().contains("ERROR: method not found"));
  }

  /**
   * Test pipeline_tempates.create api
   * @throws Exception
   */
  @Test
  public void testCreateAndGetPipelineTemplate() throws Exception {
    Arvados arv = new Arvados("arvados", "v1");

    File file = new File(getClass().getResource( "/first_pipeline.json" ).toURI());
    byte[] data = new byte[(int)file.length()];
    try {
      FileInputStream is = new FileInputStream(file);
      is.read(data);
      is.close();
    }catch(Exception e) {
      e.printStackTrace();
    }

    Map<String, Object> params = new HashMap<String, Object>();
    params.put("pipeline_template", new String(data));
    String response = arv.call("pipeline_templates", "create", params);

    JSONParser parser = new JSONParser();
    JSONObject jsonObject = (JSONObject) parser.parse(response);
    assertEquals("Expected kind to be user", "arvados#pipelineTemplate", jsonObject.get("kind"));
    String uuid = (String)jsonObject.get("uuid");
    assertNotNull("Expected uuid for pipeline template", uuid);

    // get the pipeline
    params = new HashMap<String, Object>();
    params.put("uuid", uuid);
    response = arv.call("pipeline_templates", "get", params);

    parser = new JSONParser();
    jsonObject = (JSONObject) parser.parse(response);
    assertEquals("Expected kind to be user", "arvados#pipelineTemplate", jsonObject.get("kind"));
    assertEquals("Expected uuid for pipeline template", uuid, jsonObject.get("uuid"));

    // delete the object
    params = new HashMap<String, Object>();
    params.put("uuid", uuid);
    response = arv.call("pipeline_templates", "delete", params);
  }

  /**
   * Test users.list api
   * @throws Exception
   */
  @Test
  public void testArvadosWithTokenPassed() throws Exception {
    String token = System.getenv().get("ARVADOS_API_TOKEN");
    String host = System.getenv().get("ARVADOS_API_HOST");      
    String hostInsecure = System.getenv().get("ARVADOS_API_HOST_INSECURE");

    Arvados arv = new Arvados("arvados", "v1", token, host, hostInsecure);

    Map<String, Object> params = new HashMap<String, Object>();

    String response = arv.call("users", "list", params);
    assertTrue("Expected users.list in response", response.contains("arvados#userList"));
    assertTrue("Expected users.list in response", response.contains("uuid"));

    JSONParser parser = new JSONParser();
    Object obj = parser.parse(response);
    JSONObject jsonObject = (JSONObject) obj;
    assertEquals("Expected kind to be users.list", "arvados#userList", jsonObject.get("kind"));
  }

  /**
   * Test users.list api
   * @throws Exception
   */
  @Test
  public void testCallUsersListWithLimit() throws Exception {
    Arvados arv = new Arvados("arvados", "v1");

    Map<String, Object> params = new HashMap<String, Object>();

    String response = arv.call("users", "list", params);
    assertTrue("Expected users.list in response", response.contains("arvados#userList"));
    assertTrue("Expected users.list in response", response.contains("uuid"));

    JSONParser parser = new JSONParser();
    Object obj = parser.parse(response);
    JSONObject jsonObject = (JSONObject) obj;

    assertEquals("Expected kind to be users.list", "arvados#userList", jsonObject.get("kind"));

    List items = (List)jsonObject.get("items");
    assertNotNull("expected users list items", items);
    assertTrue("expected at least one item in users list", items.size()>0);

    int numUsersListItems = items.size();

    // make the request again with limit
    params = new HashMap<String, Object>();
    params.put("limit", numUsersListItems-1);

    response = arv.call("users", "list", params);

    parser = new JSONParser();
    obj = parser.parse(response);
    jsonObject = (JSONObject) obj;

    assertEquals("Expected kind to be users.list", "arvados#userList", jsonObject.get("kind"));

    items = (List)jsonObject.get("items");
    assertNotNull("expected users list items", items);
    assertTrue("expected at least one item in users list", items.size()>0);

    int numUsersListItems2 = items.size();
    assertEquals ("Got more users than requested", numUsersListItems-1, numUsersListItems2);
  }

  @Test
  public void testGetLinksWithFilters() throws Exception {
    Arvados arv = new Arvados("arvados", "v1");

    Map<String, Object> params = new HashMap<String, Object>();

    String response = arv.call("links", "list", params);
    assertTrue("Expected links.list in response", response.contains("arvados#linkList"));

    String[] filters = new String[3];
    filters[0] = "name";
    filters[1] = "is_a";
    filters[2] = "can_manage";
    
    params.put("filters", filters);
    
    response = arv.call("links", "list", params);
    
    assertTrue("Expected links.list in response", response.contains("arvados#linkList"));
    assertFalse("Expected no can_manage in response", response.contains("\"name\":\"can_manage\""));
  }

  @Test
  public void testGetLinksWithFiltersAsList() throws Exception {
    Arvados arv = new Arvados("arvados", "v1");

    Map<String, Object> params = new HashMap<String, Object>();

    String response = arv.call("links", "list", params);
    assertTrue("Expected links.list in response", response.contains("arvados#linkList"));

    List<String> filters = new ArrayList<String>();
    filters.add("name");
    filters.add("is_a");
    filters.add("can_manage");
    
    params.put("filters", filters);
    
    response = arv.call("links", "list", params);
    
    assertTrue("Expected links.list in response", response.contains("arvados#linkList"));
    assertFalse("Expected no can_manage in response", response.contains("\"name\":\"can_manage\""));
  }

  @Test
  public void testGetLinksWithWhereClause() throws Exception {
    Arvados arv = new Arvados("arvados", "v1");

    Map<String, Object> params = new HashMap<String, Object>();

    String response = arv.call("links", "list", params);
    assertTrue("Expected links.list in response", response.contains("arvados#linkList"));

    Map<String, String> where = new HashMap<String, String>();
    where.put("where", "updated_at > '2014-05-01'");
    
    params.put("where", where);
    
    response = arv.call("links", "list", params);
    
    assertTrue("Expected links.list in response", response.contains("arvados#linkList"));
  }

}