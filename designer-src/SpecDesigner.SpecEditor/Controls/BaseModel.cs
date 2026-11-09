using System;
using System.ComponentModel;
using SpecDesignerCommon;

namespace SpecDesigner.SpecEditor.Controls
{
	// Token: 0x0200001C RID: 28
	public class BaseModel : INotifyPropertyChanged
	{
		// Token: 0x17000012 RID: 18
		// (get) Token: 0x060000C0 RID: 192 RVA: 0x00008274 File Offset: 0x00006474
		// (set) Token: 0x060000C1 RID: 193 RVA: 0x0000827C File Offset: 0x0000647C
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x060000C2 RID: 194 RVA: 0x00008285 File Offset: 0x00006485
		public BaseModel(PackageKey programKey)
		{
			this.ProgramKey = programKey;
		}

		// Token: 0x14000003 RID: 3
		// (add) Token: 0x060000C3 RID: 195 RVA: 0x00008294 File Offset: 0x00006494
		// (remove) Token: 0x060000C4 RID: 196 RVA: 0x000082CC File Offset: 0x000064CC
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060000C5 RID: 197 RVA: 0x00008301 File Offset: 0x00006501
		public void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(""));
			}
		}
	}
}
