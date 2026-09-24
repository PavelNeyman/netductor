class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.81"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.81/netductor-darwin-arm64"
      sha256 "9789421c8ec0296ef2e9e042f34ec4277626a9165ae7b65056fa0fdf54662af6"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.81/netductor-darwin-amd64"
      sha256 "f97961753bf3ecdeeeebcaf0ab3dfd1c47155adab186e99a3c3c4efbbf4c6a7c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.81/netductor-linux-amd64"
      sha256 "6cb46db95023e4270b1e974d9aeb5296235018a682e63e81af9a884c4d7bb687"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.81/netductor-linux-arm64"
      sha256 "f5ec6429b26036ada6f27467421ddf74a146fd8a08d47f3bb99a7a3feba12275"
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
