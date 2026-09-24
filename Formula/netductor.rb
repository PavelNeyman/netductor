class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.80"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.80/netductor-darwin-arm64"
      sha256 "5fdc1f8a7e25649a75a59ddb18f771930aa1502b568dc98cd92995df25dc9c7e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.80/netductor-darwin-amd64"
      sha256 "04bc16dce918cc1de50f0eb9681e3c44d63ec47c88e6a845e9420ab245095338"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.80/netductor-linux-amd64"
      sha256 "16f05c109602acb217f707b31398b3c921c1b3bd699ed9985d39f59021cc4011"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.80/netductor-linux-arm64"
      sha256 "1cd1e90b75fb7d312c9d1bb64c8e039764ba862aace04f321a55ab06cc72a463"
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
