class Wilson < Formula
  version "${ARGS}"
  desc "Wilson the task runner"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson_darwin"
  sha256 "3f23cd07749e4e806e961ded34d9a8b53d9430f6"

  def install
    bin.install "wilson"
  end
end
