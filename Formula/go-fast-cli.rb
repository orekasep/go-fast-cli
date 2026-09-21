class GoFastCli < Formula
  desc "Command-line tool to test internet speed using Fast.com with a live TUI"
  homepage "https://github.com/orekasep/go-fast-cli"
  version "1.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/orekasep/go-fast-cli/releases/download/v1.1.0/go-fast-cli_v1.1.0_darwin-arm64.tar.gz"
      sha256 "dac821854102274493945a23d4f914118d0ce6869a4c63acda4c97aab6e9c396"

      def install
        bin.install "fast" => "go-fast-cli"
        bin.install_symlink bin/"go-fast-cli" => "fast"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/orekasep/go-fast-cli/releases/download/v1.1.0/go-fast-cli_v1.1.0_darwin-amd64.tar.gz"
      sha256 "499edab0801faf74f43ab0a32682f52ca73a78a86c2d61c3ab5d225bdd976da2"

      def install
        bin.install "fast" => "go-fast-cli"
        bin.install_symlink bin/"go-fast-cli" => "fast"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/orekasep/go-fast-cli/releases/download/v1.1.0/go-fast-cli_v1.1.0_linux-arm64.tar.gz"
      sha256 "d0523a841575b75d9ab7dcfa5299c9326896dde000559a36bbaf1879091f0f9b"

      def install
        bin.install "fast" => "go-fast-cli"
        bin.install_symlink bin/"go-fast-cli" => "fast"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/orekasep/go-fast-cli/releases/download/v1.1.0/go-fast-cli_v1.1.0_linux-amd64.tar.gz"
      sha256 "dc4b9853a98bdcfb8eea00719ec0057891c44403563eb8050c798ca311b4aa13"

      def install
        bin.install "fast" => "go-fast-cli"
        bin.install_symlink bin/"go-fast-cli" => "fast"
      end
    end
  end

  test do
    assert_match "go-fast-cli", shell_output("#{bin}/go-fast-cli -version")
  end
end
