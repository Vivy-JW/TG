using System;
using System.Windows.Input;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x02000052 RID: 82
	public static class ServerLogCommands
	{
		// Token: 0x04000108 RID: 264
		public static RoutedCommand SaveCommand = new RoutedCommand("SaveCommand", typeof(ServerLogCommands));

		// Token: 0x04000109 RID: 265
		public static RoutedCommand ClearCommand = new RoutedCommand("ClearCommand", typeof(ServerLogCommands));
	}
}
