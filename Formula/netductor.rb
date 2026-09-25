class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.10"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.10/netductor-op-darwin-arm64"
      sha256 "b4f3ba38b64a4358b99da9b2c0a21967e5ae16c053e8b6d9afe9e844386612f9"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.10/netductor-op-darwin-amd64"
      sha256 "2243f064dea8f468786be701e77988363c7418f2d8e7f872e0fb8775f70a4d6b"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.10/netductor-op-linux-amd64"
      sha256 "d882420c89766e437c66ce33ebffd0ec3e727355ea2f134b110d5ea0b914a87f"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.10/netductor-op-linux-arm64"
      sha256 "a0a99d73e0aeb094211bb2d301bc8649399248de9f9861c4d6e63a7929d1ffcd"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
