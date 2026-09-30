class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.137"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.137/netductor-op-darwin-arm64"
      sha256 "e852c3d0cb57945cf7154a736ca0d6ea9c982ae50a6d2f2e86b96234f05ef70f"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.137/netductor-op-linux-amd64"
      sha256 "6d033e731d6e9ebcfa6c6a815a16b935215a5a4357dc43294d527040d0648f0f"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
