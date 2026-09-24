class Netductor < Formula
  desc "Netductor"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.93"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.93/netductor-darwin-arm64"
      sha256 "e174db64f0cc42dd3b270864f5be523ee95bd5d69013d04de4db177bb98b8c1a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.93/netductor-darwin-amd64"
      sha256 "35edfce94098d39669d029da0fa626d6651a812f2c25600bf48784a1b5c90387"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.93/netductor-linux-amd64"
      sha256 "9b44a49670714ed6d1b0b54219b7fe51df3f6245f6f4fcd6a83c10795239f6bc"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.93/netductor-linux-arm64"
      sha256 "c4a389f356831744cdac03112ef868d77e64484fe555574ae8a75d4c7a96118d"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
