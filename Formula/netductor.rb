class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.147"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.147/netductor-op-darwin-arm64"
      sha256 "5dccf2d3e690e226a851e0340da161196f503abbaaeba9e59a7ac4fcaf5a8a61"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.147/netductor-op-darwin-amd64"
      sha256 "ed0d039b905b6aa52a27c38ac80e8bfde4db5021b97731c9fe2747010b6a9613"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.147/netductor-op-linux-amd64"
      sha256 "663e6b9e7be0a9b1369d49a768d94b1ac2b9c055f8f6d56b36d185695be2fa7f"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
