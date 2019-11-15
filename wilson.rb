class Wilson < Formula
  version "0.1.0-beta.1"
  desc "Wilson the task runner"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson-darwin-amd64.tar.gz"
  sha256 "3f7d78372050558142accc1e334260889ca7c0176f2a0b36c95ddc4104386550"

  def install
    bin.install "bin/wilson_darwin_amd64"
  end
end
