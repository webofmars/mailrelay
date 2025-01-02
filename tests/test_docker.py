import os
import pytest
from testcontainers.core.image import DockerImage
from testcontainers.core.container import DockerContainer
from testcontainers.core.waiting_utils import wait_for_logs

def test_docker_build():
  """build and test the main container"""
  root = os.path.abspath(os.path.join(os.path.dirname(__file__), '../'))
  print("Using root dir %s" % root)

  # build image
  with DockerImage(path=root, tag="test-image") as image:
    logs = image.get_logs()
    print("build logs: %s" % logs)
    # run it
    with DockerContainer(str(image)) as container:
      delay = wait_for_logs(container, "Listening on TCP 0.0.0.0:")
      assert True
