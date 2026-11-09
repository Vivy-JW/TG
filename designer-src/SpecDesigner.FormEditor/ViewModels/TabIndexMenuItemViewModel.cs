using System;
using System.Windows.Input;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.ViewModels
{
	// Token: 0x02000014 RID: 20
	public class TabIndexMenuItemViewModel
	{
		// Token: 0x17000005 RID: 5
		// (get) Token: 0x06000079 RID: 121 RVA: 0x00004DA6 File Offset: 0x00002FA6
		// (set) Token: 0x0600007A RID: 122 RVA: 0x00004DAE File Offset: 0x00002FAE
		public string DisplayName { get; set; }

		// Token: 0x17000006 RID: 6
		// (get) Token: 0x0600007B RID: 123 RVA: 0x00004DB7 File Offset: 0x00002FB7
		// (set) Token: 0x0600007C RID: 124 RVA: 0x00004DBF File Offset: 0x00002FBF
		public ICommand Command { get; set; }

		// Token: 0x17000007 RID: 7
		// (get) Token: 0x0600007D RID: 125 RVA: 0x00004DC8 File Offset: 0x00002FC8
		// (set) Token: 0x0600007E RID: 126 RVA: 0x00004DD0 File Offset: 0x00002FD0
		public object CommandParam { get; set; }

		// Token: 0x17000008 RID: 8
		// (get) Token: 0x0600007F RID: 127 RVA: 0x00004DD9 File Offset: 0x00002FD9
		// (set) Token: 0x06000080 RID: 128 RVA: 0x00004DE1 File Offset: 0x00002FE1
		public XmlElement Model { get; set; }
	}
}
