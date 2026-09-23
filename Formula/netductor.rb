class Netductor < Formula
  desc "Netductor control plane CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.61"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.61/netductor-darwin-arm64"
      sha256 "5602cfe2c751a825169e00b2072d527816129dfbe874319305aed6589368e5b9"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.61/netductor-darwin-amd64"
      sha256 "742991edfa41e7ab99d1b62dfc3db05bfe342013bb15d0a93fecbadf753701e7"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.61/netductor-linux-amd64"
      sha256 "50bcbdce5f1044f304acfae5ebc2d16a217dba309dc48324d278e427c2062a52"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.61/netductor-linux-arm64"
      sha256 "af630b9a7fe8b8f7ac60f08abeee28213a26ff63189ff537624295d836599678"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
