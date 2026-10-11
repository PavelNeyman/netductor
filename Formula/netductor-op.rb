class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.327"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.327/netductor-op-darwin-arm64"
      sha256 "fdeb5ed2f4603bad9f9ed8bcf7a74c374fef1b2809be8cd65c92f5fe37bb8c37"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.327/netductor-op-darwin-amd64"
      sha256 "a444e55be996555487031d2959cc641cd8ba67fbab74395e9ff4fbb87afac497"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.327/netductor-op-linux-amd64"
      sha256 "3a8cc6fadd036a04adb73537ccb207364ad5e2224d95ae5464cf464f5ceac495"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
