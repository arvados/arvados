from setuptools import setup, find_packages
setup(name='arvados-workflow-import',
      version='0.1',
      description='Arvados workflow import',
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      download_url="https://github.com/curoverse/arvados.git",
      license='Apache 2.0',
      packages=find_packages(),
      scripts=[
          'workflowimporter.py'
      ],
      install_requires=[
          'arvados-cwl-runner'
      ],
      zip_safe=True
      )
