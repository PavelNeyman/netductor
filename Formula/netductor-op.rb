class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.390"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.390/netductor-op-darwin-arm64"
      sha256 "2eed968ceee257264337f5c1406b5b763ee6f3acdd38694e914e262bddfb4284"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.390/netductor-op-darwin-amd64"
      sha256 "988d608360336bc019898d6ba2bca5d6306ad25546d26426bd131dbc6e6165d1"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.390/netductor-op-linux-amd64"
      sha256 "0f2ee96514d2b5c56d586b6075a8dd82eb2a74a25a3b235fd7a5e89bd128f90e"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
