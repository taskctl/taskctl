class Wilson < Formula
  version "0.1.0-beta.2"
  desc "Wilson the task runner"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson-darwin-amd64.tar.gz"
  sha256 "1f26767c2103ffc69a36ff7b86d1f48745c544782d862652b9bdeeba82b795b2"

  def install
    bin.install "bin/wilson_darwin_amd64"
  end
end
