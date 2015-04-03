import arvados
import cwltool

# Whee!

def main(arguments=None):
    api = arvados.api('v1')
    api.job_tasks().create(body={
        'job_uuid': os.environ["JOB_UUID"],
        'parameters': {
            "docker_hash": "b0ae8c8d8f58d528530499841bd0bc05968f0443767132619eb3aad59d972c5a",
            "command": ["ls", "/"],
            "environment": {}
        }}).execute()
