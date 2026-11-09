using System;
using System.Collections.ObjectModel;
using System.Windows.Input;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200011E RID: 286
	internal interface IMainCodeMenu
	{
		// Token: 0x170002A6 RID: 678
		// (get) Token: 0x06000A13 RID: 2579
		ObservableCollection<ICommand> CodeMenuCommands { get; }
	}
}
