class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.78"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.78/netductor-darwin-arm64"
      sha256 "cd251fd622f9b1f1718ba1f140f8e632c00b5fa8b5279f56892eb011baedcf84"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.78/netductor-darwin-amd64"
      sha256 "3c09232a840f119b9b249a1da21680709d25a7e9a18d69a401b36cd33c1eb6b2"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.78/netductor-linux-amd64"
      sha256 "a332916948cfd2df8c579e76711ca79a4e53b9434bf5c4550b8dff8f9d5651f3"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.78/netductor-linux-arm64"
      sha256 "9bf51cd425dc80cf41d47f7c05052354a700bcad6841a341595944a97cdfc915"
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
