import re
import urllib
import ssl

from libcloud.compute.base import NodeSize, Node, NodeDriver, NodeState

all_nodes = []

class FakeDriver(NodeDriver):
    def __init__(self, *args, **kwargs):
        self.name = "FakeDriver"

    def list_sizes(self, **kwargs):
        return [NodeSize("Standard_D3", "Standard_D3", 3500, 200, 0, 0, self),
                NodeSize("Standard_D4", "Standard_D4", 7000, 400, 0, 0, self)]

    def list_nodes(self, **kwargs):
        return all_nodes

    def create_node(self, name=None,
                    size=None,
                    image=None,
                    auth=None,
                    ex_storage_account=None,
                    ex_customdata=None,
                    ex_resource_group=None,
                    ex_user_name=None,
                    ex_tags=None,
                    ex_network=None):
        global all_nodes
        all_nodes.append(Node(name, name, NodeState.RUNNING, [], [], self, size=size, extra={"tags": ex_tags}))
        ping_url = re.search(r"echo '(.*)' > /var/tmp/arv-node-data/arv-ping-url", ex_customdata).groups(1)[0] + "&instance_id=" + name
        ctx = ssl.SSLContext(ssl.PROTOCOL_SSLv23)
        ctx.verify_mode = ssl.CERT_NONE
        f = urllib.urlopen(ping_url, "", context=ctx)
        f.close()
        return all_nodes[-1]

    def destroy_node(self, cloud_node):
        global all_nodes
        all_nodes = [n for n in all_nodes if n.id != cloud_node.id]
        return True

    def get_image(self, img):
        pass

    def ex_create_tags(self, cloud_node, tags):
        pass
