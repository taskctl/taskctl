class Wilson < Formula
  version "0.1.2"
  desc "Wilson - routine tasks automation toolkit"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson-darwin-amd64.tar.gz"
  sha256 "8535cd6be8ce93371739b5dc53b265ee05af474065e88dbd2067ad84386f056b"

  def install
    bin.install "wilson_darwin_amd64"
  end
end
