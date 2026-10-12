class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.361"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.361/netductor-op-darwin-arm64"
      sha256 "a58fef207084024d6e650d13d4341a6f9987b3604f89ce994fa5f0ea57367255"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.361/netductor-op-darwin-amd64"
      sha256 "6f2a47f66d3856235e582afad71eb53afb4514789f2d5b2368995fdcbe09fb54"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.361/netductor-op-linux-amd64"
      sha256 "f718dbafa08cb7b5e214e38e3e85c90bfa5e18d42c98acfd31a8df7da21ea92c"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
