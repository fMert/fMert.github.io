# frozen_string_literal: true

require "jekyll-seo-tag"

module Jekyll
  class SeoTag
    class Drop
      def page_title
        @page_title ||= format_string(page["seo_title"] || page["title"]) || site_title
      end
    end
  end
end
