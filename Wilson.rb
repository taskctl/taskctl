class Wilson < Formula
  version "0.1.0-alpha-7"
  desc "Wilson the task runner"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson_darwin"
  sha256 "f7b90bcaccef6a7fb64a8f41a03f96fc69b12406"

  def install
    bin.install "wilson"
  end
end
