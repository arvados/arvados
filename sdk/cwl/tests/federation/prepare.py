import arvados
import json

api = arvados.api()

with open("config.json") as f:
    config = json.load(f)

for cluster_id in config["arvados_cluster_ids"]:
    for scrub_image in config["scrub_images"]:
        sp = scrub_image.split(":")
        image_name = sp[0]
        image_tag = sp[1] if len(sp) > 1 else "latest"

        search_links = api.links().list(
            filters=[['link_class', '=', 'docker_image_repo+tag'],
                     ['name', '=',
                      '{}:{}'.format(image_name, image_tag)]],
            cluster_id=cluster_id).execute()
        for s in search_links["items"]:
            print s
