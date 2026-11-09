using System;
using System.Collections.ObjectModel;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000093 RID: 147
	public class DSTableModel
	{
		// Token: 0x170001B5 RID: 437
		// (get) Token: 0x060005FD RID: 1533 RVA: 0x0001B871 File Offset: 0x00019A71
		// (set) Token: 0x060005FE RID: 1534 RVA: 0x0001B879 File Offset: 0x00019A79
		public TBLModel Table { get; set; }

		// Token: 0x170001B6 RID: 438
		// (get) Token: 0x060005FF RID: 1535 RVA: 0x0001B882 File Offset: 0x00019A82
		// (set) Token: 0x06000600 RID: 1536 RVA: 0x0001B88A File Offset: 0x00019A8A
		public ObservableCollection<DSColumnViewModel> Columns { get; set; }

		// Token: 0x06000601 RID: 1537 RVA: 0x0001B893 File Offset: 0x00019A93
		public DSTableModel()
		{
			this.Columns = new ObservableCollection<DSColumnViewModel>();
		}
	}
}
