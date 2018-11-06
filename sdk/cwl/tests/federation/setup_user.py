import arvados
import arvados.errors
import time
import json

while True:
    try:
        api = arvados.api()
        break
    except arvados.errors.ApiError:
        time.sleep(2)

existing = api.users().list(filters=[["email", "=", "test@example.com"],
                                     ["is_active", "=", True]], limit=1).execute()
if existing["items"]:
    u = existing["items"][0]
else:
    u = api.users().create(body={
        'first_name': 'Test',
        'last_name': 'User',
        'email': 'test@example.com'
    }).execute()
    api.users().activate(uuid=u["uuid"]).execute()

tok = api.api_client_authorizations().create(body={
    "owner_uuid": u["uuid"]
}).execute()

with open("cwl.output.json", "w") as f:
    json.dump({
        "test_user_uuid": u["uuid"],
        "test_user_token": tok["api_token"]
    }, f)
