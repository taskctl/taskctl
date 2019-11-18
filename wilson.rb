class Wilson < Formula
  version "0.1.1"
  desc "Wilson - routine tasks automation toolkit"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson-darwin-amd64.tar.gz"
  sha256 "76662473c7f43a962edfffe13be94e0cd5dcd0f09b82f06fc58b4f85fc2dbfc3"

  def install
    bin.install "bin/wilson_darwin_amd64"
  end
end
