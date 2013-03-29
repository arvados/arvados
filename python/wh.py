import pprint
import httplib2
import sys
import json
import ssl
from apiclient.discovery import build
import uritemplate
import os

url = ('https://localhost:3000/discovery/v1/apis/'
  'orvos/v1/rest')

api_key = "AIzaSyBqwUZG-rw6h5p9hoI0iPA4HCZZVj3v93g"
token = "4exra8rbjsfzovhfppims2gbbs7632tcgr4tsrtsta8t67a086"

# Create an httplib2.Http object to handle our HTTP requests and authorize it
# with our good Credentials.
http = httplib2.Http()
#http = credentials.authorize(http)
http.disable_ssl_certificate_validation=True

service = build('orvos', 'v1', http=http, discoveryServiceUrl=url)

#pprint.pprint(content)
#print json.dumps(response, sort_keys = True, indent = 3)
	

def getList(dr):
	for i in dir(dr):
		if not i[0] =='_':
			print "   %s" % i
	print ''

def getExecute(bd, s, s1):
	collection = getattr(bd, s)()
	request = getattr(collection, s1)(oauth_token=token)
	response = request.execute()
	return response


if len(sys.argv) == 1:
	print '\nThis Orvos Cluster supports the following api commands:\n'
	getList(service)

elif len(sys.argv) == 2:
	if sys.argv[1] in dir(service):
		print '\nThis api command supports the following method:\n'
		getList(getattr(service, sys.argv[1])())
	else:
		print '\nThis Orvos Cluster supports the following api commands:\n'
		getList(service)
		print '"%s" is not a supported api command of the Orvos Cluster.\n' % sys.argv[1]

if len(sys.argv) >= 3:
	if sys.argv[2] == 'create':
		pass
	elif sys.argv[2] == 'delete':
		pass
	elif sys.argv[2] == 'get':
		response = getExecute(service, sys.argv[1], 'list')
		if len(sys.argv) == 3 :
			print "Please specify the uuid of the user you want to get"
		if len(sys.argv) == 5 and sys.argv[3] == '--json':
			for itm in response.get('items', []):
				if itm['uuid'] == sys.argv[4]:
					print json.dumps(itm)
		if len(sys.argv) == 5 and sys.argv[3] == '--jsonhuman':
			for itm in response.get('items', []):
				if itm['uuid'] == sys.argv[4]:
					print json.dumps(itm, sort_keys=True, indent=3)
		else:
			for itm in response.get('items', []):
				if itm['uuid'] == sys.argv[3]:
					for x in itm:
						print '%s: %s' % (x, itm[x])
	elif sys.argv[2] == 'list':
		response = getExecute(service, sys.argv[1], sys.argv[2])
		if len(sys.argv) == 4 and sys.argv[3] == '--json':
			print json.dumps(response, sort_keys=True, indent = 3)
		else:
			for itm in response.get('items',[]):
				print itm['uuid']
	elif sys.argv[2] == 'update':
		pass
