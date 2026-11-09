using System;
using System.Collections;
using System.Collections.Generic;
using System.Runtime.Remoting;
using System.Runtime.Remoting.Channels;
using System.Runtime.Remoting.Channels.Ipc;
using System.Runtime.Serialization.Formatters;
using System.Threading;
using System.Windows;
using System.Windows.Threading;

namespace Microsoft.Shell
{
	// Token: 0x02000025 RID: 37
	public static class SingleInstance<TApplication> where TApplication : Application, ISingleInstanceApp
	{
		// Token: 0x17000095 RID: 149
		// (get) Token: 0x06000247 RID: 583 RVA: 0x0000AC74 File Offset: 0x00008E74
		public static IList<string> CommandLineArgs
		{
			get
			{
				return SingleInstance<TApplication>.commandLineArgs;
			}
		}

		// Token: 0x06000248 RID: 584 RVA: 0x0000AC7C File Offset: 0x00008E7C
		public static bool InitializeAsFirstInstance(string uniqueName)
		{
			SingleInstance<TApplication>.commandLineArgs = SingleInstance<TApplication>.GetCommandLineArgs(uniqueName);
			string text = uniqueName + Environment.UserName;
			string text2 = text + ":" + "SingeInstanceIPCChannel";
			bool flag;
			SingleInstance<TApplication>.singleInstanceMutex = new Mutex(true, text, out flag);
			if (flag)
			{
				SingleInstance<TApplication>.CreateRemoteService(text2);
			}
			else
			{
				SingleInstance<TApplication>.SignalFirstInstance(text2, SingleInstance<TApplication>.commandLineArgs);
			}
			return flag;
		}

		// Token: 0x06000249 RID: 585 RVA: 0x0000ACD6 File Offset: 0x00008ED6
		public static void Cleanup()
		{
			if (SingleInstance<TApplication>.singleInstanceMutex != null)
			{
				SingleInstance<TApplication>.singleInstanceMutex.Close();
				SingleInstance<TApplication>.singleInstanceMutex = null;
			}
			if (SingleInstance<TApplication>.channel != null)
			{
				ChannelServices.UnregisterChannel(SingleInstance<TApplication>.channel);
				SingleInstance<TApplication>.channel = null;
			}
		}

		// Token: 0x0600024A RID: 586 RVA: 0x0000AD08 File Offset: 0x00008F08
		public static IList<string> GetCommandLineArgs(string uniqueApplicationName)
		{
			string[] array;
			if (AppDomain.CurrentDomain.ActivationContext == null)
			{
				array = Environment.GetCommandLineArgs();
			}
			else
			{
				array = AppDomain.CurrentDomain.SetupInformation.ActivationArguments.ActivationData;
			}
			if (array == null)
			{
				array = new string[0];
			}
			return new List<string>(array);
		}

		// Token: 0x0600024B RID: 587 RVA: 0x0000AD50 File Offset: 0x00008F50
		private static void CreateRemoteService(string channelName)
		{
			BinaryServerFormatterSinkProvider binaryServerFormatterSinkProvider = new BinaryServerFormatterSinkProvider();
			binaryServerFormatterSinkProvider.TypeFilterLevel = TypeFilterLevel.Full;
			IDictionary dictionary = new Dictionary<string, string>();
			dictionary["name"] = channelName;
			dictionary["portName"] = channelName;
			dictionary["exclusiveAddressUse"] = "false";
			SingleInstance<TApplication>.channel = new IpcServerChannel(dictionary, binaryServerFormatterSinkProvider);
			ChannelServices.RegisterChannel(SingleInstance<TApplication>.channel, true);
			SingleInstance<TApplication>.IPCRemoteService ipcremoteService = new SingleInstance<TApplication>.IPCRemoteService();
			RemotingServices.Marshal(ipcremoteService, "SingleInstanceApplicationService");
		}

		// Token: 0x0600024C RID: 588 RVA: 0x0000ADC4 File Offset: 0x00008FC4
		private static void SignalFirstInstance(string channelName, IList<string> args)
		{
			IpcClientChannel ipcClientChannel = new IpcClientChannel();
			ChannelServices.RegisterChannel(ipcClientChannel, true);
			string text = "ipc://" + channelName + "/SingleInstanceApplicationService";
			SingleInstance<TApplication>.IPCRemoteService ipcremoteService = (SingleInstance<TApplication>.IPCRemoteService)RemotingServices.Connect(typeof(SingleInstance<TApplication>.IPCRemoteService), text);
			if (ipcremoteService != null)
			{
				ipcremoteService.InvokeFirstInstance(args);
			}
		}

		// Token: 0x0600024D RID: 589 RVA: 0x0000AE10 File Offset: 0x00009010
		private static object ActivateFirstInstanceCallback(object arg)
		{
			IList<string> list = arg as IList<string>;
			SingleInstance<TApplication>.ActivateFirstInstance(list);
			return null;
		}

		// Token: 0x0600024E RID: 590 RVA: 0x0000AE2C File Offset: 0x0000902C
		private static void ActivateFirstInstance(IList<string> args)
		{
			if (Application.Current == null)
			{
				return;
			}
			TApplication tapplication = (TApplication)((object)Application.Current);
			tapplication.SignalExternalCommandLineArgs(args);
		}

		// Token: 0x04000158 RID: 344
		private const string Delimiter = ":";

		// Token: 0x04000159 RID: 345
		private const string ChannelNameSuffix = "SingeInstanceIPCChannel";

		// Token: 0x0400015A RID: 346
		private const string RemoteServiceName = "SingleInstanceApplicationService";

		// Token: 0x0400015B RID: 347
		private const string IpcProtocol = "ipc://";

		// Token: 0x0400015C RID: 348
		private static Mutex singleInstanceMutex;

		// Token: 0x0400015D RID: 349
		private static IpcServerChannel channel;

		// Token: 0x0400015E RID: 350
		private static IList<string> commandLineArgs;

		// Token: 0x02000026 RID: 38
		private class IPCRemoteService : MarshalByRefObject
		{
			// Token: 0x0600024F RID: 591 RVA: 0x0000AE5B File Offset: 0x0000905B
			public void InvokeFirstInstance(IList<string> args)
			{
				if (Application.Current != null)
				{
					Application.Current.Dispatcher.BeginInvoke(DispatcherPriority.Normal, new DispatcherOperationCallback(SingleInstance<TApplication>.ActivateFirstInstanceCallback), args);
				}
			}

			// Token: 0x06000250 RID: 592 RVA: 0x0000AE83 File Offset: 0x00009083
			public override object InitializeLifetimeService()
			{
				return null;
			}
		}
	}
}
