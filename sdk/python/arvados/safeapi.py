import threading
import api
import keep
import config
import copy

class ThreadSafeApiCache(object):
    """Threadsafe wrapper for API objects.  This stores and returns a different api
    object per thread, because httplib2 which underlies apiclient is not
    threadsafe.
    """

    def __init__(self, apiconfig=None, keep_params={}):
        if apiconfig is None:
            apiconfig = config.settings()
        self.apiconfig = copy.copy(apiconfig)
        self.local = threading.local()
        self.keep = keep.KeepClient(api_client=self, **keep_params)

    def localapi(self):
        if 'api' not in self.local.__dict__:
            self.local.api = api.api('v1', False, apiconfig=self.apiconfig)
        return self.local.api

    def __getattr__(self, name):
        # Proxy nonexistent attributes to the thread-local API client.
        if name == "api_token":
            return self.apiconfig['ARVADOS_API_TOKEN']
        try:
            return getattr(self.localapi(), name)
        except AttributeError:
            return super(ThreadSafeApiCache, self).__getattr__(name)
