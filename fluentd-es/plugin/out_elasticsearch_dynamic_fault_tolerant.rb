# encoding: UTF-8
require 'fluent/plugin/out_elasticsearch_dynamic'

module Fluent::Plugin
  class ElasticsearchOutputDynamicFaultTolerant < ElasticsearchOutputDynamic

    Fluent::Plugin.register_output('elasticsearch_dynamic_fault_tolerant', self)

    def configure(conf)
      super
    end

    def write(chunk)
      super
    end

    def send_bulk(data, host, index)
      begin
        response = client(host).bulk body: data, index: index
        if response['errors']
          log.error "Could not push log to Elasticsearch: #{response}"
        end
      rescue => e
        @_es = nil if @reconnect_on_error

        # if the elastic service is not present or no elastic instance is able to handle a request
        # the idea of this if statement is to prevent clearing of queue for exceeded limit of retries 
        if !e.message.nil? &&
            (e.message.include?("getaddrinfo: Name or service not known (SocketError)") ||
                e.message.include?("connect_write timeout reached"))
            log.error "could not push logs to Elasticsearch cluster (#{connection_options_description(host)}): #{e.message}"
            return
        end
        
        # FIXME: identify unrecoverable errors and raise UnrecoverableRequestFailure instead
        raise RecoverableRequestFailure, "could not push logs to Elasticsearch cluster (#{connection_options_description(host)}): #{e.message}"
      end
    end
  end
end
