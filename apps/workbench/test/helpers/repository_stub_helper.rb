module RepositoryStubHelper
  # Supply some fake git content.
  def stub_repo_content opts={}
    fakesha1 = opts[:sha1] || 'abcdefabcdefabcdefabcdefabcdefabcdefabcd'
    fakefilename = opts[:filename] || 'COPYING'
    fakefilesrc = File.expand_path('../../../../../'+fakefilename, __FILE__)
    fakefile = File.read fakefilesrc
    fakecommit = <<-EOS
      commit abcdefabcdefabcdefabcdefabcdefabcdefabcd
      Author: Fake R <fake@example.com>
      Date:   Wed Apr 1 11:59:59 2015 -0400

          It's a fake commit.

    EOS
    Repository.any_instance.stubs(:ls_tree_lr).with(fakesha1).returns <<-EOS
      100644 blob eec475862e6ec2a87554e0fca90697e87f441bf5     226    .gitignore
      100644 blob acbd7523ed49f01217874965aa3180cccec89d61     625    COPYING
      100644 blob d645695673349e3947e8e5ae42332d0ac3164cd7   11358    LICENSE-2.0.txt
      100644 blob c7a36c355b4a2b94dfab45c9748330022a788c91     622    README
      100644 blob dba13ed2ddf783ee8118c6a581dbf75305f816a3   34520    agpl-3.0.txt
      100644 blob 9bef02bbfda670595750fd99a4461005ce5b8f12     695    apps/workbench/.gitignore
      100644 blob b51f674d90f68bfb50d9304068f915e42b04aea4    2249    apps/workbench/Gemfile
      100644 blob b51f674d90f68bfb50d9304068f915e42b04aea4    2249    apps/workbench/Gemfile
      100755 blob cdd5ebaff27781f93ab85e484410c0ce9e97770f    1012    crunch_scripts/hash
    EOS
    Repository.any_instance.
      stubs(:cat_file).with(fakesha1, fakefilename).returns fakefile
    Repository.any_instance.
      stubs(:show).with(fakesha1).returns fakecommit
    return fakesha1, fakecommit, fakefile
  end
end
