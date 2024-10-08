function add_tag_to_record(tag, timestamp, record)
      record["tag"] = tag
      return 1, timestamp, record
end