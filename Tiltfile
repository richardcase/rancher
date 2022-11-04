# -*- mode: Python -*-

# Originally based on the Tiltfile from the Cluster API project

load("./tilt/project/Tiltfile", "project_enable")
load("./tilt/io/Tiltfile", "info", "warn", "file_write")


# set defaults
version_settings(True, ">=0.22.2")

settings = {
    "enable_projects": [],
    "kind_cluster_name": os.getenv("kind_CLUSTER_NAME", "rancher-dev"),
    "debug": {},
}

# global settings
tilt_file = "./tilt-settings.yaml" if os.path.exists("./tilt-settings.yaml") else "./tilt-settings.json"
settings.update(read_yaml(
    tilt_file,
    default = {},
))

os.putenv("kind_CLUSTER_NAME", settings.get("kind_cluster_name"))

allow_k8s_contexts(settings.get("allowed_contexts"))

os_name = str(local("go env GOOS")).rstrip("\n")
os_arch = str(local("go env GOARCH")).rstrip("\n")

if settings.get("trigger_mode") == "manual":
    trigger_mode(TRIGGER_MODE_MANUAL)

if settings.get("default_registry") != "":
    default_registry(settings.get("default_registry"))

always_enable_projects = ["manager"]

projects = {
    "manager": {
        "context": ".",
        "image": "rancher/rancher",
        "live_reload_deps": [
            "main.go",
            "go.mod",
            "go.sum",
            "pkg",
        ],
        "chart": {
            "dir": "chart",
            "values": [
                "ingress.enabled=false",
                "replicas=1",
                "tls=external",
                "postDelete.enabled=false",
                "debug=false"
            ],
            "container_name": "rancher"
        },
        "env": {
            "CATTLE_DEV_MODE": "30",
            "CATTLE_FEATURES": "embedded-cluster-api=true",
            "CATTLE_SYSTEM_CHART_DEFAULT_BRANCH": "dev-v2.7"
        },
        "label": "rancher"
    }
}

# Users may define their own Tilt customizations in tilt.d. This directory is excluded from git and these files will
# not be checked in to version control.
def include_user_tilt_files():
    user_tiltfiles = listdir("tilt.d")
    for f in user_tiltfiles:
        include(f)

def enable_projects():
    for name in get_projects():
        p = projects.get(name)
        project_enable(name, p, settings.get("debug").get(name, {}))

def get_projects():
    user_enable_projects = settings.get("enable_projects", [])
    return {k: "" for k in user_enable_projects + always_enable_projects}.keys()

# Reads a projects tilt-project.json file and merges it into the projects map.
# A list of dictionaries is also supported by enclosing it in brackets []
# An example file looks like this:
# {
#     "name": "aws",
#     "config": {
#         "image": "gcr.io/k8s-staging-cluster-api-aws/cluster-api-aws-controller",
#         "live_reload_deps": [
#             "main.go", "go.mod", "go.sum", "api", "cmd", "controllers", "pkg"
#         ]
#     }
# }

def load_project_tiltfiles():
    project_repos = settings.get("project_repos", [])

    for repo in project_repos:
        file = repo + "/tilt-project.yaml" if os.path.exists(repo + "/tilt-project.yaml") else repo + "/tilt-project.json"
        if not os.path.exists(file):
            fail("Failed to load provider. No tilt-project.{yaml|json} file found in " + repo)
        project_details = read_yaml(file, default = {})
        if type(project_details) != type([]):
            project_details = [project_details]
        for item in project_details:
            project_name = item["name"]
            project_config = item["config"]
            if "context" in project_config:
                project_config["context"] = repo + "/" + project_config["context"]
            else:
                project_config["context"] = repo
            if "go_main" not in project_config:
                project_config["go_main"] = "main.go"
            projects[project_name] = project_config



##############################
# Actual work happens here
##############################

include_user_tilt_files()

load_project_tiltfiles()

enable_projects()
