class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.157"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.157/netductor-op-darwin-arm64"
      sha256 "ea07a0e258027105de92b85d8932addf8c31d5517178ca6e97ddc4927c053b50"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.157/netductor-op-darwin-amd64"
      sha256 "55e8a0b1ee46a71d18eb98b14b25c98bd64d16a08c57788cfff1c5e5248410d7"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.157/netductor-op-linux-amd64"
      sha256 "1a3d592ad9383937a3f46d8ed8a40942f93f0b8c620eaa404b21f1510f724b13"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
