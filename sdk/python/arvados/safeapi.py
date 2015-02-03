import threading
import api
import keep
import config

class SafeApi(object):
    """Threadsafe wrapper for API object.  This stores and returns a different api
    object per thread, because httplib2 which underlies apiclient is not
    threadsafe.
    """

    def __init__(self, apiconfig=None, keep_params={}):
        if not apiconfig:
            apiconfig = config
        self.host = apiconfig.get('ARVADOS_API_HOST')
        self.api_token = apiconfig.get('ARVADOS_API_TOKEN')
        self.insecure = apiconfig.flag_is_true('ARVADOS_API_HOST_INSECURE')
        self.local = threading.local()
        self.keep = keep.KeepClient(api_client=self, **keep_params)

    def localapi(self):
        if 'api' not in self.local.__dict__:
            self.local.api = api.api('v1', False, self.host,
                                         self.api_token, self.insecure)
        return self.local.api

    def __getattr__(self, name):
        # Proxy nonexistent attributes to the thread-local API client.
        try:
            return getattr(self.localapi(), name)
        except AttributeError:
            return super(SafeApi, self).__getattr__(name)
