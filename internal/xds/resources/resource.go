package resources

import (
	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	v31 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	filedaccesslogv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/proxy_protocol/v3"
	proxy_protocolv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/proxy_protocol/v3"
	tcpproxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	v33 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/ptypes"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"lb/apis/v1alpha1"
	"log"
	"time"
)

func MakeCluster(clusterName string, connectTimeout time.Duration, health v1alpha1.HealthCheck, maglevTableSize uint64, healthPanicThreshold float32, hashBalanceFactory uint32, endpoints []Endpoint) *cluster.Cluster {

	healthCheck := &core.HealthCheck{
		Timeout:            durationpb.New(health.Timeout),                            //1초동안 응답이 없으면, 헬스체크 실패
		Interval:           durationpb.New(health.Interval),                           // 헬스 체크 요청 간격
		UnhealthyThreshold: &wrapperspb.UInt32Value{Value: health.UnhealthyThreshold}, //서비스 제외 전 헬스체크 횟수
		HealthyThreshold:   &wrapperspb.UInt32Value{Value: health.HealthyThreshold},   // 복귀하기 위한 헬스체크 성공 횟수
		HealthChecker: &core.HealthCheck_HttpHealthCheck_{ // HTTP 기반
			HttpHealthCheck: &core.HealthCheck_HttpHealthCheck{
				Path: health.HttpHealthCheck.Path,
			},
		},
	}

	return &cluster.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       ptypes.DurationProto(connectTimeout),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		LbPolicy:             cluster.Cluster_MAGLEV,
		CommonLbConfig: &cluster.Cluster_CommonLbConfig{
			HealthyPanicThreshold: &v33.Percent{Value: float64(healthPanicThreshold)},
			ConsistentHashingLbConfig: &cluster.Cluster_CommonLbConfig_ConsistentHashingLbConfig{
				UseHostnameForHashing: false,
				HashBalanceFactor:     &wrappers.UInt32Value{Value: hashBalanceFactory},
			},
		},
		LoadAssignment: MakeEndpoint(clusterName, endpoints),
		LbConfig: &cluster.Cluster_MaglevLbConfig_{
			MaglevLbConfig: &cluster.Cluster_MaglevLbConfig{TableSize: wrapperspb.UInt64(maglevTableSize)},
		},
		HealthChecks:     []*core.HealthCheck{healthCheck},
		DnsLookupFamily:  cluster.Cluster_V4_ONLY,
		EdsClusterConfig: makeEDSCluster(),
	}
}

func makeEDSCluster() *cluster.Cluster_EdsClusterConfig {
	return &cluster.Cluster_EdsClusterConfig{
		EdsConfig: makeConfigSource(),
	}
}

func MakeHTTPListener(listenerName, address string, port uint32, accessLogPath string, chains []v1alpha1.FilterChain) *listener.Listener {
	filter := chains[0].Filters[0]

	return &listener.Listener{
		Name: listenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  address,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		},
		ListenerFilters: []*listener.ListenerFilter{
			{
				Name: "envoy.filters.listener.proxy_protocol",
				ConfigType: &listener.ListenerFilter_TypedConfig{
					TypedConfig: mustMarshalAny(&proxy_protocolv3.ProxyProtocol{}),
				},
			},
			//{
			//	Name: "envoy.filters.listener.original_src",
			//	ConfigType: &listener.ListenerFilter_TypedConfig{
			//		TypedConfig: mustMarshalAny(&originalsrcv3.OriginalSrc{}),
			//	},
			//},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{
				{
					Name: filter.Name,
					ConfigType: &listener.Filter_TypedConfig{
						TypedConfig: mustMarshalAny(&tcpproxy.TcpProxy{
							StatPrefix: filter.TypeConfig.StatPrefix,
							ClusterSpecifier: &tcpproxy.TcpProxy_Cluster{
								Cluster: filter.TypeConfig.Cluster,
							},
							AccessLog: []*v31.AccessLog{
								{
									Name: "envoy.access_loggers.file",
									ConfigType: &accesslogv3.AccessLog_TypedConfig{
										TypedConfig: mustMarshalAny(&filedaccesslogv3.FileAccessLog{
											Path: accessLogPath,
											AccessLogFormat: &filedaccesslogv3.FileAccessLog_LogFormat{
												LogFormat: &core.SubstitutionFormatString{
													Format: &core.SubstitutionFormatString_JsonFormat{
														JsonFormat: &_struct.Struct{
															Fields: map[string]*structpb.Value{
																"authority":                         structpb.NewStringValue("%REQ(:AUTHORITY)%"),
																"bytes_received":                    structpb.NewStringValue("%BYTES_RECEIVED%"),
																"bytes_sent":                        structpb.NewStringValue("%BYTES_SENT%"),
																"connection_termination_details":    structpb.NewStringValue("%CONNECTION_TERMINATION_DETAILS%"),
																"downstream_local_address":          structpb.NewStringValue("%DOWNSTREAM_LOCAL_ADDRESS%"),
																"downstream_remote_address":         structpb.NewStringValue("%DOWNSTREAM_REMOTE_ADDRESS%"),
																"duration":                          structpb.NewStringValue("%DURATION%"),
																"method":                            structpb.NewStringValue("%REQ(:METHOD)%"),
																"path":                              structpb.NewStringValue("%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%"),
																"protocol":                          structpb.NewStringValue("%PROTOCOL%"),
																"request_id":                        structpb.NewStringValue("%REQ(X-REQUEST-ID)%"),
																"requested_server_name":             structpb.NewStringValue("%REQUESTED_SERVER_NAME%"),
																"response_code":                     structpb.NewStringValue("%RESPONSE_CODE%"),
																"response_code_details":             structpb.NewStringValue("%RESPONSE_CODE_DETAILS%"),
																"response_flags":                    structpb.NewStringValue("%RESPONSE_FLAGS%"),
																"route_name":                        structpb.NewStringValue("%ROUTE_NAME%"),
																"start_time":                        structpb.NewStringValue("%START_TIME%"),
																"upstream_cluster":                  structpb.NewStringValue("%UPSTREAM_CLUSTER%"),
																"upstream_host":                     structpb.NewStringValue("%UPSTREAM_HOST%"),
																"upstream_local_address":            structpb.NewStringValue("%UPSTREAM_LOCAL_ADDRESS%"),
																"upstream_service_time":             structpb.NewStringValue("%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%"),
																"upstream_transport_failure_reason": structpb.NewStringValue("%UPSTREAM_TRANSPORT_FAILURE_REASON%"),
																"user_agent":                        structpb.NewStringValue("%REQ(USER-AGENT)%"),
																"x_forwarded_for":                   structpb.NewStringValue("%REQ(X-FORWARDED-FOR)%"),
															},
														},
													},
												},
											},
										}),
									},
								},
							},
							HashPolicy: []*v33.HashPolicy{
								{
									PolicySpecifier: &v33.HashPolicy_SourceIp_{},
								},
							},
						}),
					},
				},
			},
		}},
	}
}

func mustMarshalAny(pb proto.Message) *anypb.Any {
	a, err := anypb.New(pb)
	if err != nil {
		log.Fatalf("failed to marshal proto message %v: %v", pb, err)
	}
	return a
}

func makeConfigSource() *core.ConfigSource {
	source := &core.ConfigSource{}
	source.ResourceApiVersion = resource.DefaultAPIVersion
	source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
		ApiConfigSource: &core.ApiConfigSource{
			TransportApiVersion:       resource.DefaultAPIVersion,
			ApiType:                   core.ApiConfigSource_GRPC,
			SetNodeOnFirstMessageOnly: true,
			GrpcServices: []*core.GrpcService{{
				TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: "xds_cluster"},
				},
			}},
		},
	}
	return source
}

func MakeEndpoint(clusterName string, eps []Endpoint) *endpoint.ClusterLoadAssignment {
	var endpoints []*endpoint.LbEndpoint

	for _, e := range eps {
		endpoints = append(endpoints, &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.SocketAddress_TCP,
								Address:  e.UpstreamHost,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: e.UpstreamPort,
								},
							},
						},
					},
				},
			},
		})
	}

	return &endpoint.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: endpoints,
		}},
	}
}
