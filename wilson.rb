class Wilson < Formula
  version "0.1.0-beta.4"
  desc "Wilson the task runner"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson-darwin-amd64.tar.gz"
  sha256 "314dfbaf87186d56353f9049da88e791425c8e966f8763947d056b71adfa1bd3"

  def install
    bin.install "bin/wilson_darwin_amd64"
  end
end
