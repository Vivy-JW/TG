using System;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000009 RID: 9
	public abstract class AbstractUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x1700000F RID: 15
		// (get) Token: 0x06000051 RID: 81 RVA: 0x00003767 File Offset: 0x00001967
		// (set) Token: 0x06000052 RID: 82 RVA: 0x0000376F File Offset: 0x0000196F
		internal XmlElement Element { get; set; }

		// Token: 0x06000053 RID: 83 RVA: 0x00003778 File Offset: 0x00001978
		public AbstractUndoRedoCommand(XmlElement element)
		{
			this.Element = element;
		}

		// Token: 0x06000054 RID: 84 RVA: 0x00003787 File Offset: 0x00001987
		public virtual void Undo()
		{
			throw new NotImplementedException();
		}

		// Token: 0x06000055 RID: 85 RVA: 0x0000378E File Offset: 0x0000198E
		public virtual void Execute()
		{
			throw new NotImplementedException();
		}

		// Token: 0x06000056 RID: 86 RVA: 0x00003795 File Offset: 0x00001995
		public virtual void Clear()
		{
			this.Element = null;
		}

		// Token: 0x06000057 RID: 87 RVA: 0x0000379E File Offset: 0x0000199E
		internal SpecificationInfo GetSpecificationInfo()
		{
			if (this.Element != null)
			{
				return SettingManager.Get().GetTzpManger(this.Element.Key).SpecificationInfo;
			}
			return null;
		}
	}
}
