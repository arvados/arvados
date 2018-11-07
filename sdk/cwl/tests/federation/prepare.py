import arvados

api = arvados.api()

print(api.users().current().execute())
