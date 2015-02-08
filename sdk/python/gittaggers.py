from setuptools.command.egg_info import egg_info
import subprocess
import time

class GitTagger(egg_info):
    """Tag the build with git commit info.

    Exact choice and format is determined by subclass's tags_to_add
    method.

    If a build tag has already been set (e.g., "egg_info -b", building
    from source package), leave it alone.
    """
    def git_commit_info(self):
        gitinfo = subprocess.check_output(
            ['git', 'log', '--first-parent', '--max-count=1',
             '--format=format:%ct %h', '.']).split()
        assert len(gitinfo) == 2
        return {
            'commit_utc': time.strftime(
                '%Y%m%d%H%M%S', time.gmtime(int(gitinfo[0]))),
            'commit_sha1': gitinfo[1],
        }

    def tags(self):
        if self.tag_build is None:
            self.tag_build = self.tags_to_add()
        return egg_info.tags(self)


class TagBuildWithCommitDateAndSha1(GitTagger):
    """Tag the build with the sha1 and date of the last git commit."""
    def tags_to_add(self):
        return '.{commit_utc}+{commit_sha1}'.format(**self.git_commit_info())


class TagBuildWithCommitDate(GitTagger):
    """Tag the build with the date of the last git commit."""
    def tags_to_add(self):
        return '.{commit_utc}'.format(**self.git_commit_info())
