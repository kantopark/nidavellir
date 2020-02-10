import re
from pathlib import Path
from subprocess import Popen, PIPE


def build_all_dockerfiles():
    folder = Path(__file__).parent
    for file in folder.iterdir():
        if file.name.endswith('.Dockerfile'):
            build_image(file.as_posix())


def build_image(file: str):
    dockerfile, image_name = form_image_details(file)
    if image_name is None:
        print(f"Could not derive image name and tag, check {file}")
        return

    code = run_process("docker", "image", "build", "-t", image_name, "-f", dockerfile, ".")
    if code != 0:
        print(f"Failed to build {dockerfile}")
        return

    latest = image_name.split(":")[0] + ":latest"
    run_process("docker", "image", "tag", image_name, latest)
    run_process("docker", "image", "prune", "-f")


def run_process(*args: str):
    print(' '.join(args))
    command = Popen(args, stdout=PIPE, universal_newlines=True, shell=True)

    while True:
        output = command.stdout.readline()
        if not output and command.poll() is not None:
            return command.poll()
        if output:
            print(output.strip(), flush=True)


def form_image_details(file: str):
    with open(file) as f:
        for line in f:
            match = re.match(r"FROM (\w+):([\d.]+)", line)
            if match is not None:
                lang = match.group(1)
                return f"{lang}.Dockerfile", f"danielbok/nida-{lang}:{match.group(2)}"

    return None


if __name__ == '__main__':
    build_all_dockerfiles()
